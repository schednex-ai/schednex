/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
*/

package main

import (
	"context"
	"flag"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/schednex.git/pkg/k8sgpt_client"
	"github.com/k8sgpt-ai/schednex.git/pkg/placement"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
)

func main() {
	// Setup Zap logger with development mode
	var enableDevelopmentMode bool
	var allowAI bool
	flag.BoolVar(&enableDevelopmentMode, "development", true, "Enable development mode for Zap logger")
	flag.BoolVar(&allowAI, "allow-ai", true, "Enable AI for scheduling")
	// Try to get the kubeconfig from outside the cluster (for development)
	flag.Parse()

	// Initialize the Zap logger using controller-runtime
	logger := zap.New(zap.UseDevMode(enableDevelopmentMode))
	ctrl.SetLogger(logger)
	log := ctrl.Log.WithName("Schednex")

	// Try to use in-cluster config if available, fallback to kubeconfig
	var config *rest.Config
	var err error

	if config, err = rest.InClusterConfig(); err != nil {
		// Fallback to kubeconfig from outside the cluster
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			log.Error(err, "Failed to build Kubernetes config")
			os.Exit(1)
		}
	}

	v1alpha1.AddToScheme(scheme.Scheme)

	// Create a new client for interacting with low-level Kubernetes API
	ctrlclient, err := client.New(config, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Error(err, "Failed to create Kubernetes client")
		os.Exit(1)
	}

	// Create a Kubernetes Client using client-go
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err, "Failed to create Kubernetes clientset")
		os.Exit(1)
	}

	// Create a new k8sgpt Client
	k8sgptClient, err := k8sgpt_client.NewClient(ctrlclient, log)
	if err != nil {
		log.Error(err, "Failed to connect k8sgpt client")
		os.Exit(1)
	}

	// Create a new metrics Client
	metricsClient, err := metricsv.NewForConfig(config)
	if err != nil {
		log.Error(err, "Failed to create metrics client")
		os.Exit(1)
	}
	// Create a new Placement Coordinator
	coordinator := placement.NewCoordinator(k8sgptClient, clientset, metricsClient, log)
	// Start custom scheduler loop
	for {
		// List unscheduled pods
		unscheduledPods, err := getUnscheduledPods(clientset)
		if err != nil {
			log.Error(err, "Failed to list unscheduled pods")
			time.Sleep(10 * time.Second)
			continue
		}

		for _, pod := range unscheduledPods.Items {
			log.Info("Scheduling Pod", "namespace", pod.Namespace, "name", pod.Name)

			node, err := coordinator.FindNodeForPod(pod, allowAI)
			if err != nil {
				log.Error(err, "Failed to find a node for pod", "namespace", pod.Namespace, "name", pod.Name)
				continue
			}

			err = bindPodToNode(clientset, pod, node)
			if err != nil {
				log.Error(err, "Failed to bind pod to node", "pod", pod.Name, "node", node)
			} else {
				log.Info("Successfully scheduled pod", "pod", pod.Name, "node", node)
			}
		}

		// Sleep before the next scheduling loop
		time.Sleep(10 * time.Second)
	}
}

// getUnscheduledPods lists all pods that are not yet scheduled
func getUnscheduledPods(clientset *kubernetes.Clientset) (*v1.PodList, error) {
	return clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName==",
	})
}

// bindPodToNode binds a pod to a specific node
func bindPodToNode(clientset *kubernetes.Clientset, pod v1.Pod, nodeName string) error {
	binding := &v1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: pod.Namespace,
			Name:      pod.Name,
		},
		Target: v1.ObjectReference{
			Kind: "Node",
			Name: nodeName,
		},
	}

	return clientset.CoreV1().Pods(pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
}
