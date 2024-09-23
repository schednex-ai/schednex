<img src="images/logo.png" width="400">

A custom Kubernetes scheduler that uses insights from K8sGPT, powered by AI. 
Enabling the smartest placement of your workloads.

## Requirements

- K8sGPT-operator installed with a deployed K8sGPT custom resource
- Metrics Server installed in the cluster:
  - `kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml`

## Installation

- Install K8sGPT operator with [this guide](https://github.com/k8sgpt-ai/k8sgpt-operator?tab=readme-ov-file#installation)
  

- Install the scheduler:

_Until this repository is public, you can use the following command to install the scheduler:_
```
git clone https://github.com/k8sgpt-ai/schednex.git
make deploy
