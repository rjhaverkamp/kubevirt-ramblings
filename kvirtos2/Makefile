.PHONY: build run test clean docker-build deploy undeploy port-forward tidy

# Build the application
build:
	go build -o bin/kvirtos2 .

# Run locally
run:
	go run main.go --kubeconfig ~/.kube/config

# Test the application
test:
	go test ./...

# Download and tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t ghcr.io/rjhaverkamp/dbridge:kvirtos2-v1-api .

docker-push:
	docker push ghcr.io/rjhaverkamp/dbridge:kvirtos2-v1-api

# Deploy to Kubernetes
deploy:
	kubectl apply -f deploy.yaml

# Remove from Kubernetes
undeploy:
	kubectl delete -f deploy.yaml

# Port forward for local access
port-forward:
	kubectl port-forward svc/kvirtos2 8080:8080

# Check deployment status
status:
	kubectl get pods -l app=kvirtos2
	kubectl get svc kvirtos2

# View logs
logs:
	kubectl logs -l app=kvirtos2 -f

# Load Docker image into kind cluster (if using kind)
kind-load:
	kind load docker-image kvirtos2:latest
