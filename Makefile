# Variables
# IMAGE_NAME = ghcr.io/schednex-ai/schednex
IMAGE_NAME = ghcr.io/schednex-ai/schednex
TAG = v1.3.0 # x-release-please-version
CONFIG_DIR = config
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

.DEFAULT_GOAL = all

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: tidy
tidy: ## Run go mod tidy
	go mod tidy

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

##@ Build

.PHONY: all
all: tidy fmt build ##

.PHONY: build
build: ## Build the Docker image
	docker buildx build -t $(IMAGE_NAME):$(TAG) . --platform="linux/arm64,linux/amd64"  --push

.PHONY: push
push: ## Push the Docker image to the repository
	docker push $(IMAGE_NAME):$(TAG)

.PHONY: clean
clean: # Clean up any local images
	docker rmi $(IMAGE_NAME):$(TAG) || true

##@ Deployment

.PHONY: cluster-up
cluster-up: ## Create a development cluster with kind
	kind create cluster --config examples/kind-three-node-cluster.yaml
	kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
	kubectl -n kube-system patch deployment metrics-server \
	  --type='json' \
	  -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'

.PHONY: deploy
deploy: helm ## Deploy the Kubernetes manifests from the config folder
	$(HELM) install schednex ./charts/schednex

.PHONY: undeploy
undeploy: helm ## Undeploy the Kubernetes manifests
	$(HELM) uninstall schednex

##@ Tools

.PHONY: bin
bin: ## Create bin directory
	mkdir -p $(shell pwd)/bin

HELM = $(shell pwd)/bin/helm-$(GOOS)-$(GOARCH)
HELM_VERSION ?= v3.16.0

.PHONY: helm
helm: bin ## Download helm binary if it doesn't exist
	@[ -f $(HELM) ] || { \
	set -e ;\
	curl -L https://get.helm.sh/helm-$(HELM_VERSION)-$(GOOS)-$(GOARCH).tar.gz | tar xz; \
	mv $(GOOS)-$(GOARCH)/helm $(HELM); \
	chmod +x $(HELM); \
	rm -rf ./$(GOOS)-$(GOARCH)/; \
	}
