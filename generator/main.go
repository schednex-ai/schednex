package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const podPrefix = "example-pod-"

// createPod creates a pod using the Kubernetes client
func createPod(clientset *kubernetes.Clientset, podName string) {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: v1.PodSpec{
			SchedulerName: "schednex",
			Containers: []v1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	_, err := clientset.CoreV1().Pods("default").Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Error creating pod %s: %v", podName, err)
	}
	fmt.Printf("Created pod: %s\n", podName)
}

// deletePods deletes all pods with the name prefix "example-pod-"
func deletePods(clientset *kubernetes.Clientset) {
	// List all pods in the "default" namespace
	podList, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error listing pods: %v", err)
	}

	for _, pod := range podList.Items {
		if strings.HasPrefix(pod.Name, podPrefix) {
			err = clientset.CoreV1().Pods("default").Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
			if err != nil {
				log.Printf("Error deleting pod %s: %v", pod.Name, err)
			} else {
				fmt.Printf("Deleted pod: %s\n", pod.Name)
			}
		}
	}
}

func main() {
	// Parse flags
	deleteFlag := flag.Bool("delete", false, "Delete all example pods")
	flag.Parse()

	kubeconfig := getKubeConfigPath()

	// Build the config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error loading kubeconfig: %v", err)
	}

	// Create a clientset to interact with Kubernetes
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
	}

	if *deleteFlag {
		// Delete all example pods
		deletePods(clientset)
	} else {
		// Generate 100 pods with a 1-second delay between each pod
		for i := 1; i <= 100; i++ {
			podName := fmt.Sprintf("%s%d", podPrefix, i)
			createPod(clientset, podName)
			time.Sleep(1 * time.Second) // Add 1-second delay between pod creations
		}
	}
}

// getKubeConfigPath finds the kubeconfig file in the user's home directory or uses the KUBECONFIG environment variable
func getKubeConfigPath() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %v", err)
	}
	return filepath.Join(home, ".kube", "config")
}
