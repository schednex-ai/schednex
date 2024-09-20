/*
Copyright 2023 The K8sGPT Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package k8sgpt_client

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	cntrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Client for communicating with the K8sGPT in cluster deployment
type Client struct {
	conn *grpc.ClientConn
}

func (c *Client) Close() error {
	return c.conn.Close()
}

// NewClient will detect K8sGPT instances currently running in the Kubernetes cluster and connect to the first it finds
func NewClient(ctrlruntimeClient cntrlclient.Client) (*Client, error) {

	// add log
	log.Printf("Creating new client for K8sGPT")
	k8sgptList := &v1alpha1.K8sGPTList{}
	err := ctrlruntimeClient.List(context.Background(), k8sgptList, &cntrlclient.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list K8sGPT objects: %v", err)
	}
	// how many items
	log.Printf("Number of K8sGPT objects found: %d", len(k8sgptList.Items))
	// Check list length
	if len(k8sgptList.Items) == 0 {
		return nil, fmt.Errorf("no K8sGPT objects found")
	}
	// Get the first K8sGPT Object (for now)
	k8sgptConfig := k8sgptList.Items[0]
	// print the raw object
	log.Printf("K8sGPT object: %v", k8sgptConfig)
	// Generate address
	log.Printf("Generating address for K8sGPT")
	address, err := GenerateAddress(context.Background(), ctrlruntimeClient, &k8sgptConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %v", err)
	}
	// print address to log
	log.Printf("Address: %s", address)
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create context: %v", err)
	}
	client := &Client{conn: conn}

	return client, nil
}

func GenerateAddress(ctx context.Context, cli cntrlclient.Client, k8sgptConfig *v1alpha1.K8sGPT) (string, error) {
	var address string
	var ip net.IP

	// Get service IP and port for k8sgpt-deployment
	svc := &corev1.Service{}
	err := cli.Get(ctx, cntrlclient.ObjectKey{Namespace: k8sgptConfig.Namespace,
		Name: k8sgptConfig.Name}, svc)
	if err != nil {
		return "", err
	}
	ip = net.ParseIP(svc.Spec.ClusterIP)
	if ip.To4() != nil {
		address = fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)
	} else {
		address = fmt.Sprintf("[%s]:%d", svc.Spec.ClusterIP, svc.Spec.Ports[0].Port)
	}

	fmt.Printf("Creating new client for %s\n", address)
	// Test if the port is open
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	fmt.Printf("Connection established between %s and localhost with time out of %d seconds.\n", address, int64(1))
	fmt.Printf("Remote Address : %s \n", conn.RemoteAddr().String())

	return address, nil
}
