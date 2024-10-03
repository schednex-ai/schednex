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
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/k8sgpt-ai/schednex/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/schednex/pkg/k8sgpt_client"
	"github.com/k8sgpt-ai/schednex/pkg/placement"
	"golang.org/x/time/rate"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// Setup Zap logger with development mode
	var enableDevelopmentMode bool
	var allowAI bool
	var metricsAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableDevelopmentMode, "development", true, "Enable development mode for Zap logger")
	flag.BoolVar(&allowAI, "allow-ai", true, "Enable AI for scheduling")
	// Try to get the kubeconfig from outside the cluster (for development)
	flag.Parse()
	// Initialize the Zap logger using controller-runtime
	logger := zap.New(zap.UseDevMode(enableDevelopmentMode))
	ctrl.SetLogger(logger)
	log := ctrl.Log.WithName("Schednex")

	// Add metrics
	if os.Getenv("LOCAL_MODE") != "" {
		min := 7000
		max := 8000
		metricsAddr = fmt.Sprintf("%d", rand.Intn(max-min+1)+min)
	}
	metricsBuilder := metrics.InitializeMetrics()
	// Start the metrics handler
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info("Starting metrics server", "port", metricsAddr)
		err := http.ListenAndServe(fmt.Sprintf(":%s", metricsAddr), nil)
		if err != nil {
			log.Error(err, "Error starting metrics server")
			return
		}
	}()

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
	if err = v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		log.Error(err, "Failed to add to scheme")
		os.Exit(1)
	}

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
	k8sgptClient, err := k8sgpt_client.NewClient(ctrlclient, metricsBuilder, log)
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
	coordinator := placement.NewCoordinator(k8sgptClient, clientset,
		metricsClient, metricsBuilder,
		log)

	parentCtx := context.Background()
	ctx, _ := context.WithCancel(parentCtx)

	// Create rate limiter
	rateLimiter := rate.NewLimiter(1, 1)

	// Start custom scheduler loop
	log.Info("Starting Schednex...")
	for {
		err := rateLimiter.Wait(ctx)
		if err != nil {
			log.Error(err, "Rate limiter error")
			return
		}
		// List unscheduled pods
		unscheduledPods, err := getUnscheduledPods(clientset)
		if err != nil {
			log.Error(err, "Failed to list unscheduled pods")
			//simple back off
			time.Sleep(10 * time.Second)
			continue
		}

		for _, pod := range unscheduledPods.Items {
			log.Info("Scheduling Pod", "namespace", pod.Namespace, "name", pod.Name)

			node, err := coordinator.FindNodeForPod(pod, allowAI)
			if err != nil {
				// print the error we get back
				log.Error(err, "Error from K8sGPT", "pod", pod.Name)
				continue
			}

			err = bindPodToNode(clientset, pod, node)
			if err != nil {
				log.Error(err, "Failed to bind pod to node", "pod", pod.Name, "node", node)
			} else {
				log.Info("Successfully scheduled pod", "pod", pod.Name, "node", node)
			}
			event := &v1.Event{
				ObjectMeta: metav1.ObjectMeta{
					// Use a unique name
					Name:      fmt.Sprintf("Scheduled %s-%s", pod.Name, pod.UID),
					Namespace: pod.Namespace,
				},
			}
			// Set the event message
			event.Message = "Pod " + pod.Name + " scheduled on node " + node
			// Set the event type
			event.Type = "Normal"
			// Set the event reason
			event.Reason = "Scheduled"
			// Set the involved object#
			event.InvolvedObject = v1.ObjectReference{
				Kind:      "Pod",
				Namespace: pod.Namespace,
				Name:      pod.Name,
				UID:       pod.UID,
			}
			// Source should be the schednex operator
			event.Source = v1.EventSource{
				Component: "Schednex",
			}
			// Send the event
			_, err = clientset.CoreV1().Events(pod.Namespace).Create(context.TODO(), event, metav1.CreateOptions{})
			if err != nil {
				log.Error(err, "Failed to send event", "pod", pod.Name)
			}
		}
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
