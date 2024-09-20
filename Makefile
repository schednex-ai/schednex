# Variables
IMAGE_NAME = k8sgpt/schednex
TAG = v0
CONFIG_DIR = config

# Default target
all: build

# Build the Docker image
build:
	docker build -t $(IMAGE_NAME):$(TAG) .

update-image-tag:
	@echo "Updating image tag to $(IMAGE_NAME):$(TAG) in $(DEPLOYMENT_FILE)..."
	sed -i.bak "s|\(image: $(IMAGE_NAME):\).*|\1$(TAG)|" $(DEPLOYMENT_FILE)
	@echo "Updated image tag in $(DEPLOYMENT_FILE)."

# Push the Docker image to the repository
push:
	docker push $(IMAGE_NAME):$(TAG)

# Clean up any local images
clean:
	docker rmi $(IMAGE_NAME):$(TAG) || true

# Deploy the Kubernetes manifests
deploy:
	kubectl apply -f $(CONFIG_DIR)

undeploy:
	kubectl delete -f $(CONFIG_DIR) 
# Help command
help:
	@echo "Makefile commands:"
	@echo "  make build   - Build the Docker image for Schednex."
	@echo "  make push    - Push the Docker image to the repository."
	@echo "  make clean   - Remove the local Docker image."
	@echo "  make deploy   - Deploy the Kubernetes manifests from the config folder."
	@echo "  make help    - Show this help message."
