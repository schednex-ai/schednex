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
	"log"
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
	_, err := c.kubernetesClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
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
		log.Fatalf("Failed to get node metrics: %v", err)
		return "", err
	}
	// Flatten nodeMetricsList items into json
	nodeMetricsListJson, err := json.Marshal(nodeMetricsList)
	if err != nil {
		return "", err
	}
	// Print nodeMetricsListJson
	log.Printf("NodeMetricsList: %s", nodeMetricsListJson)
	// Simple logic: select the first available node (custom logic can go here)
	var prompt string = "Given the following nodes and analysis of issues in the cluster, I want you to return only the node name for placement, nothing else." +
		"Please find the data in two segments below: \n" +
		"1. Nodes in the cluster: %s\n" +
		"2. Analysis of issues in the cluster (this may be empty) %s\n"
	combinedPrompt := fmt.Sprintf(prompt, nodeMetricsListJson, k8sgptAnalysis)
	// Combine the k8sgpt Analysis and the node metrics to make a decision
	// Print the combined prompt
	log.Printf("Combined Prompt: %s", combinedPrompt)
	// Send query
	response, err := c.k8sgptClient.Query(combinedPrompt)
	if err != nil {
		return "", err
	}

	// Print the response
	log.Printf("Response: %s", response)
	// if response is a single word only use it
	if len(response) == 1 {
		return response, nil
	}
	return "", fmt.Errorf("no nodes available")
}
