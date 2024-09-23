<img src="images/logo.png" width="400">

A custom Kubernetes scheduler that uses insights from K8sGPT, powered by AI. 
Enabling the smartest placement of your workloads.

## Requirements

- K8sGPT-operator installed with a deployed K8sGPT custom resource
- Metrics Server installed in the cluster:
  - `kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml`

## Installation

- Install K8sGPT operator on your cluster:
```
helm repo add k8sgpt https://charts.k8sgpt.ai/
helm repo update
helm install release k8sgpt/k8sgpt-operator -n k8sgpt-operator-system --create-namespace
```
- Add an OpenAI secret and create the K8sGPT CR:

_Token from ENV in this example_
```
kubectl create secret generic k8sgpt-sample-secret --from-literal=openai-api-key=$OPENAI_TOKEN -n k8sgpt-operator-system
```
_Minimal K8sGPT CR_
```
kubectl apply -f - << EOF
apiVersion: core.k8sgpt.ai/v1alpha1
kind: K8sGPT
metadata:
  name: k8sgpt-sample
  namespace: k8sgpt-operator-system
spec:
  ai:
    enabled: true
    model: gpt-3.5-turbo
    backend: openai
    secret:
      name: k8sgpt-sample-secret
      key: openai-api-key
  noCache: false
  repository: ghcr.io/k8sgpt-ai/k8sgpt
  version: v0.3.41
EOF
```
- Install the scheduler:

_Until this repository is public, you can use the following command to install the scheduler:_
```
git clone https://github.com/k8sgpt-ai/schednex.git
make deploy
