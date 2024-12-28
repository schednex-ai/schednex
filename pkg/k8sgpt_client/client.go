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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	rpc "buf.build/gen/go/k8sgpt-ai/k8sgpt/grpc/go/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/k8sgpt-ai/k8sgpt/protocolbuffers/go/schema/v1"
	"github.com/cenkalti/backoff/v4"
	_ "github.com/cenkalti/backoff/v4"
	"github.com/go-logr/logr"
	"github.com/k8sgpt-ai/k8sgpt-operator/api/v1alpha1"
	"github.com/k8sgpt-ai/schednex/pkg/backoff_config"
	"github.com/k8sgpt-ai/schednex/pkg/metrics"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	cntrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Client for communicating with the K8sGPT in cluster deployment
type Client struct {
	conn                *grpc.ClientConn
	currentK8sgptConfig v1alpha1.K8sGPT
	metrics             *metrics.MetricBuilder
	log                 logr.Logger
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetCurrentConfig() (v1alpha1.K8sGPT, error) {
	return c.currentK8sgptConfig, nil
}

// NewClient will detect K8sGPT instances currently running in the Kubernetes cluster and connect to the first it finds
func NewClient(ctrlruntimeClient cntrlclient.Client, m *metrics.MetricBuilder, log logr.Logger) (*Client, error) {

	log = log.WithName("k8sgpt-client")

	k8sgptList := &v1alpha1.K8sGPTList{}

	getK8sGPTObject := func() error {
		log.Info("Waiting for K8sGPT Custom Resources")
		err := ctrlruntimeClient.List(context.Background(), k8sgptList, &cntrlclient.ListOptions{})
		if err != nil {
			objectBackoffCounter := m.GetCounterVec("schednex_k8sgpt_object_backoff")
			if objectBackoffCounter != nil {
				objectBackoffCounter.WithLabelValues("k8sgpt", "backoff").Inc()
			}
			return err
		}
		if len(k8sgptList.Items) == 0 {
			return fmt.Errorf("no K8sGPT objects found")
		}
		return nil
	}
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.MaxElapsedTime = backoff_config.MAX_TIME
	backoffConfig.MaxInterval = backoff_config.MAX_INTERVAL

	err := backoff.Retry(getK8sGPTObject, backoffConfig)
	if err != nil {
		log.Error(err, "Failed to list K8sGPT objects")
	}

	// Check list length
	if len(k8sgptList.Items) == 0 {
		return nil, fmt.Errorf("no K8sGPT objects found")
	}

	// TODO: Get the first K8sGPT Object (for now)
	k8sgptConfig := k8sgptList.Items[0]

	// Generate address
	log.Info("Generating address for K8sGPT")
	address, err := GenerateAddress(context.Background(), ctrlruntimeClient, &k8sgptConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address: %v", err)
	}
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to create context: %v", err)
	}
	client := &Client{conn: conn,
		currentK8sgptConfig: k8sgptConfig,
		metrics:             m,
	}

	return client, nil
}

func GenerateAddress(ctx context.Context, cli cntrlclient.Client, k8sgptConfig *v1alpha1.K8sGPT) (string, error) {
	var address string
	var ip net.IP

	if os.Getenv("LOCAL_MODE") != "" {
		return "localhost:8080", nil
	}
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

func (c *Client) RunAnalysis(allowAIRequest bool) (string, error) {
	config, err := c.GetCurrentConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get current config: %v", err)
	}
	c.log.Info("Running analysis for K8sGPT")
	client := rpc.NewServerAnalyzerServiceClient(c.conn)
	req := &schemav1.AnalyzeRequest{
		Explain:   config.Spec.AI.Enabled && allowAIRequest,
		Nocache:   config.Spec.NoCache,
		Backend:   config.Spec.AI.Backend,
		Namespace: config.Spec.TargetNamespace,
		Filters:   config.Spec.Filters,
		Anonymize: *config.Spec.AI.Anonymize,
		Language:  config.Spec.AI.Language,
	}
	res, err := client.Analyze(context.Background(), req)
	if err != nil {
		interconnectBackoff := c.metrics.GetCounterVec("schednex_k8sgpt_interconnect_backoff")
		if interconnectBackoff != nil {
			interconnectBackoff.WithLabelValues("k8sgpt", "interconnect").Inc()
		}
		return "", fmt.Errorf("failed to call Analyze RPC: %v", err)
	}
	c.log.Info("Analysis complete")
	// convert to a json structure for searchability
	jsonBytes, err := json.Marshal(res.Results)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func (c *Client) Query(prompt string) (string, error) {
	config, err := c.GetCurrentConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get current config: %v", err)
	}
	c.log.Info("Running query for K8sGPT")
	client := rpc.NewServerQueryServiceClient(c.conn)
	req := &schemav1.QueryRequest{
		Query:   prompt,
		Backend: config.Spec.AI.Backend,
	}

	res, err := client.Query(context.Background(), req)
	if err != nil {
		c.metrics.GetCounterVec("schednex_k8sgpt_interconnect_backoff").WithLabelValues("k8sgpt", "backoff").Inc()
		return "", fmt.Errorf("failed to call Query RPC: %v", err)
	}
	c.log.Info("Query complete")
	if res.Error.Message != "" {
		return "", fmt.Errorf("error in query response: %s", res.Error.Message)
	}
	return res.Response, nil
}
