package placement

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/k8sgpt-ai/schednex.git/pkg/k8sgpt_client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"strings"
)

type Coordinator struct {
	kubernetesClient *kubernetes.Clientset
	k8sgptClient     *k8sgpt_client.Client
	metricsClient    *metricsv.Clientset
	log              logr.Logger
}

func NewCoordinator(client *k8sgpt_client.Client, kubernetesClient *kubernetes.Clientset,
	metricsClient *metricsv.Clientset, log logr.Logger) *Coordinator {
	// print creating coordinator
	log.Info("Creating coordinator")
	return &Coordinator{
		kubernetesClient: kubernetesClient,
		k8sgptClient:     client,
		metricsClient:    metricsClient,
		log:              log.WithName("coordinator"),
	}
}
func (c *Coordinator) FindNodeForPod(pod v1.Pod, allowAI bool) (string, error) {
	k8sgptAnalysis, err := c.k8sgptClient.RunAnalysis(allowAI)
	if err != nil {
		c.log.Error(err, "Something went wrong with K8sGPT analysis")
		return "", err
	}
	// Look at the current analysis results and the load on the current nodes
	// Get node metrics
	nodeMetricsList, err := c.metricsClient.MetricsV1beta1().
		NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		c.log.Error(err, "Failed to get node metrics")
		return "", err
	}
	// Flatten nodeMetricsList items into json
	nodeMetricsListJson, err := json.Marshal(nodeMetricsList)
	if err != nil {
		return "", err
	}
	// Print nodeMetricsListJson
	// Simple logic: select the first available node (custom logic can go here)
	var prompt string = "Given the following nodes and analysis of issues in the cluster, I want you to tell me the best node for placement, no other text." +
		"Please find the data in two segments below: \n" +
		"1. Nodes in the cluster: %s\n" +
		"2. Analysis of issues in the cluster (this may be empty) %s\n"
	combinedPrompt := fmt.Sprintf(prompt, nodeMetricsListJson, k8sgptAnalysis)
	// Combine the K8sGPT Analysis and the node metrics to make a decision
	// Send query
	response, err := c.k8sgptClient.Query(combinedPrompt)
	if err != nil {
		return "", err
	}
	// Often the response can be a list of multiple nodes, sometimes even missing a string seperator
	fmt.Printf("Response: %s\n", response)
	firstResponse := strings.Split(response, " ")[0]
	if firstResponse == "" {
		return "", fmt.Errorf("no response found")
	}
	// Loop through the first response and make sure it matches exactly to a node name
	for _, node := range nodeMetricsList.Items {
		if firstResponse == node.Name {
			return firstResponse, nil
		}
	}
	return "", fmt.Errorf("node not found")
}
