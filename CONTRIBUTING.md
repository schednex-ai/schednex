## Contributing

## Architecture

Schednex is a drop in replacement for the Kubernetes default scheduler.
It will only be utilised by workloads that name `schedulerName: Schednex`

You can see the architecture of Schednex below:

<img src="images/diagram.svg" width="400"/>

### Getting setup

1. Clone this repository and K8sGPT
2. Install the test cluster with `kind` and the command `make cluster-up`
3. Run K8sGPT locally e.g. `go run main.go serve`
4. Run the scheduler locally e.g. `LOCAL_MODE=on go run main.go`
5. Test it with a pod e.g. `kubectl apply -f examples/example-pod.yaml`