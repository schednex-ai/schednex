<img src="images/logo.png" width="400">

A custom Kubernetes scheduler that uses insights from K8sGPT, powered by AI. 
Enabling the smartest placement of your workloads.

## Requirements

- K8sGPT-operator installed with a deployed K8sGPT custom resource
- Metrics Server installed in the cluster:
  - `kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml`
