package placement

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/k8sgpt-ai/schednex/pkg/k8sgpt_client"
	"github.com/k8sgpt-ai/schednex/pkg/metrics"
	"github.com/k8sgpt-ai/schednex/pkg/prompt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Coordinator struct {
	kubernetesClient *kubernetes.Clientset
	k8sgptClient     *k8sgpt_client.Client
	metricsClient    *metricsv.Clientset
	metricsBuilder   *metrics.MetricBuilder
	log              logr.Logger
}

func NewCoordinator(client *k8sgpt_client.Client, kubernetesClient *kubernetes.Clientset,
	metricsClient *metricsv.Clientset, metricsBuilder *metrics.MetricBuilder, log logr.Logger) *Coordinator {
	// print creating coordinator
	log.Info("Creating coordinator")
	return &Coordinator{
		kubernetesClient: kubernetesClient,
		k8sgptClient:     client,
		metricsClient:    metricsClient,
		log:              log.WithName("coordinator"),
		metricsBuilder:   metricsBuilder,
	}
}

func (c *Coordinator) findNodePlacement(pod v1.Pod) (string, error) {
	// Find the node the pod is currently placed on
	nodeName := pod.Spec.NodeName
	if nodeName != "" {
		return nodeName, nil
	}
	// If the pod is not placed on a node, return an error
	return "", fmt.Errorf("pod is not placed on a node")
}

func (c *Coordinator) findRelatives(pod v1.Pod) ([]v1.Pod, error) {

	deduplicate := func(pods []v1.Pod) []v1.Pod {
		keys := make(map[string]bool)
		list := []v1.Pod{}
		for _, pod := range pods {
			if _, value := keys[pod.Name]; !value {
				keys[pod.Name] = true
				list = append(list, pod)
			}
		}
		return list
	}

	// find if the pod is in a replica set by looking at the owner references
	for _, r := range pod.OwnerReferences {
		// Look at all types of controllers...
		switch r.Kind {
		case "ReplicaSet":
			// Get the replica set
			replicaSet, err := c.kubernetesClient.AppsV1().ReplicaSets(pod.Namespace).Get(context.TODO(), r.Name, metav1.GetOptions{})
			if err != nil {
				c.log.Error(err, "Failed to get replica set")
				return nil, err
			}
			// Find out what nodes the other pods are placed on
			// Get the pods in the replica set
			pods, err := c.kubernetesClient.CoreV1().Pods(pod.Namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("pod-template-hash=%s", replicaSet.Labels["pod-template-hash"]),
			})
			if err != nil {
				c.log.Error(err, "Failed to get pods in replica set")
				return nil, err
			}
			return deduplicate(pods.Items), nil
		case "StatefulSet":
			// Get the statefulset and other pods
			statefulSet, err := c.kubernetesClient.AppsV1().StatefulSets(pod.Namespace).Get(context.TODO(), r.Name, metav1.GetOptions{})
			if err != nil {
				c.log.Error(err, "Failed to get stateful set")
				return nil, err
			}
			// Find out what nodes the other pods are placed on
			// Get the pods in the stateful set
			pods, err := c.kubernetesClient.CoreV1().Pods(pod.Namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("statefulset.kubernetes.io/pod-name=%s", statefulSet.Name),
			})
			if err != nil {
				c.log.Error(err, "Failed to get pods in stateful set")
				return nil, err
			}
			return deduplicate(pods.Items), nil
		case "DaemonSet":
			// Get the daemonset and other pods
			daemonSet, err := c.kubernetesClient.AppsV1().DaemonSets(pod.Namespace).Get(context.TODO(), r.Name, metav1.GetOptions{})
			if err != nil {
				c.log.Error(err, "Failed to get daemon set")
				return nil, err
			}
			// Find out what nodes the other pods are placed on
			// Get the pods in the daemon set
			pods, err := c.kubernetesClient.CoreV1().Pods(pod.Namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("app=%s", daemonSet.Labels["app"]),
			})
			if err != nil {
				c.log.Error(err, "Failed to get pods in daemon set")
				return nil, err
			}
			return deduplicate(pods.Items), nil
		}
	}
	return nil, nil
}
func (c *Coordinator) FindNodeForPod(pod v1.Pod, allowAI bool) (string, error) {
	k8sgptAnalysis, err := c.k8sgptClient.RunAnalysis(allowAI)
	if err != nil {
		c.log.Error(err, "Something went wrong with K8sGPT analysis")
		return "", err
	}
	// Get node metrics
	nodeMetricsList, err := c.metricsClient.MetricsV1beta1().
		NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		c.log.Error(err, "Failed to get node metrics")
		return "", err
	}
	// Flatten nodeMetricsList items into JSON
	nodeMetricsListJson, err := json.Marshal(nodeMetricsList)
	if err != nil {
		return "", err
	}
	// Get relatives that have a different name from the current pod
	relatives, err := c.findRelatives(pod)
	if err != nil {
		c.log.Error(err, "Failed to get relatives")
	}
	relative_placement := make(map[string]string)
	for _, relative := range relatives {
		if relative.Name != pod.Name {
			nodeName, err := c.findNodePlacement(relative)
			if err != nil {
				continue
			}
			relative_placement[fmt.Sprintf("%s is a related pod and resides on", relative.Name)] = nodeName
		}
	}
	// Formulate the prompt
	combinedPrompt := fmt.Sprintf(prompt.Standard, nodeMetricsListJson, k8sgptAnalysis, relative_placement)
	// Send the query to the AI backend
	response, err := c.k8sgptClient.Query(combinedPrompt)
	if err != nil {
		return "", err
	}
	// Print the raw response for debugging
	fmt.Printf("Response: %s\n", response)
	// Trim response to avoid trailing/leading spaces or newlines
	response = strings.TrimSpace(response)
	// Direct match check
	for _, node := range nodeMetricsList.Items {
		if response == node.Name {
			podsScheduledCounter := c.metricsBuilder.GetCounterVec("schednex_pods_scheduled")
			if podsScheduledCounter != nil {
				podsScheduledCounter.WithLabelValues("schednex").Inc()
			}
			return response, nil
		}
	}
	// Fallback to regex extraction if direct match fails
	re := regexp.MustCompile(`\b(\S+)\b`)
	matches := re.FindStringSubmatch(response)
	if len(matches) > 1 {
		firstResponse := matches[1]
		// Validate extracted node name
		for _, node := range nodeMetricsList.Items {
			if firstResponse == node.Name {
				podsScheduledCounter := c.metricsBuilder.GetCounterVec("schednex_pods_scheduled")
				if podsScheduledCounter != nil {
					podsScheduledCounter.WithLabelValues("schednex").Inc()
				}
				return firstResponse, nil
			}
		}
	}
	// Increment failure counter if no match found
	placementFailureCounter := c.metricsBuilder.GetCounterVec("schednex_placement_failure")
	if placementFailureCounter != nil {
		placementFailureCounter.WithLabelValues("schednex", "placement").Inc()
	}
	// Fallback to default scheduler
	c.log.Info("Delegating to default scheduler")
	patchData := []byte(`{"spec": {"schedulerName": null}}`)
	_, err = c.kubernetesClient.CoreV1().Pods(pod.Namespace).Patch(
		context.TODO(),
		pod.Name,
		types.StrategicMergePatchType,
		patchData,
		metav1.PatchOptions{},
	)
	if err != nil {
		c.log.Error(err, "Failed to patch pod to use default scheduler")
		return "", err
	}
	return "", fmt.Errorf("no node found")
}
