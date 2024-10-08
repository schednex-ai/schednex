<img src="images/logo.png" width="400">


![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/schednex-ai/schednex/build_container.yml)
![GitHub Release](https://img.shields.io/github/v/release/schednex-ai/schednex)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/schednex-ai/schednex)
![GitHub License](https://img.shields.io/github/license/schednex-ai/schednex)
![GitHub last commit (branch)](https://img.shields.io/github/last-commit/schednex-ai/schednex/main)


Schednex enables the smartest placement of your workloads by drawing on telemetry from K8sGPT and context awareness from AI.

## Who is this for?

- Cluster Operators who want dynamic workloads and avoid hot spots.
- SRE who want to ensure additional cluster resiliency and ability to operate through partial outage.
- Platform engineers who want to enable individual cluster tenants to manage their scheduling decisions.

## How it works

Schednex is a Kubernetes scheduler that uses insights from K8sGPT to make intelligent decisions about where to place your workloads. It uses the K8sGPT API to get recommendations for the best node to place a pod based on the pod's requirements and the current state of the cluster.
If it cannot make the decision in a timely fashion it will leverage the default scheduler, and always enable a placement decision.

## Requirements

- k8sgpt-operator installed with a deployed K8sGPT custom resource
  - k8sgpt resource must have AI enabled and using the `openai` backend
  - **K8sGPT v0.3.41 and later**
  - Disable caching `noCache: true` within the K8sGPT CR
- Metrics Server installed in the cluster:
  - `kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml`

## Installation
- Install K8sGPT operator with [this guide](https://github.com/k8sgpt-ai/k8sgpt-operator?tab=readme-ov-file#installation)

- Install the scheduler:
```
helm repo add schednex-ai https://charts.schednex.ai
helm repo update
helm install schednex-scheduler schednex-ai/schednex -n kube-system
```

## Usage

If you wish Schednex to takeover from the default schedular, add the following annotation to your pod spec:
```
schedulerName: schednex
```
