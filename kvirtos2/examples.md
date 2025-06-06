# kvirtos2 Examples

This document provides examples of how to build, deploy, and use kvirtos2.

## Building and Running

### Local Development

```bash
# Download dependencies
go mod tidy

# Run locally (requires kubeconfig)
go run main.go --kubeconfig ~/.kube/config --namespace default --port 8080
```

### Building Docker Image

```bash
# Build the Docker image
docker build -t kvirtos2:latest .

# Or build for specific architecture
docker buildx build --platform linux/amd64 -t kvirtos2:latest .
```

### Deploying to Kubernetes

```bash
# Apply the deployment manifests
kubectl apply -f deploy.yaml

# Check deployment status
kubectl get pods -l app=kvirtos2

# Check service
kubectl get svc kvirtos2

# Port forward for local access
kubectl port-forward svc/kvirtos2 8080:8080
```

## API Usage Examples

### Create a Virtual Machine

```bash
curl -X POST http://localhost:8080/v2.1/servers \
  -H "Content-Type: application/json" \
  -d '{
    "server": {
      "name": "test-vm",
      "imageRef": "fedora-cloud",
      "flavorRef": "m1.small",
      "metadata": {
        "purpose": "testing",
        "owner": "admin"
      }
    }
  }'
```

### List Virtual Machines

```bash
curl http://localhost:8080/v2.1/servers
```

### Get VM Details

```bash
curl http://localhost:8080/v2.1/servers/{vm-id}
```

### Start a VM

```bash
curl -X POST http://localhost:8080/v2.1/servers/{vm-id}/action \
  -H "Content-Type: application/json" \
  -d '{"os-start": null}'
```

### Stop a VM

```bash
curl -X POST http://localhost:8080/v2.1/servers/{vm-id}/action \
  -H "Content-Type: application/json" \
  -d '{"os-stop": null}'
```

### Reboot a VM

```bash
curl -X POST http://localhost:8080/v2.1/servers/{vm-id}/action \
  -H "Content-Type: application/json" \
  -d '{"reboot": {"type": "SOFT"}}'
```

### Delete a VM

```bash
curl -X DELETE http://localhost:8080/v2.1/servers/{vm-id}
```

## Available Flavors

The following flavors are predefined:

- `m1.tiny`: 0.5 CPU, 512MB RAM
- `m1.small`: 1 CPU, 2GB RAM  
- `m1.medium`: 2 CPU, 4GB RAM
- `m1.large`: 4 CPU, 8GB RAM

## Available Images

The following images are available:

- `ubuntu-20.04`: Ubuntu 20.04 container disk
- `fedora-cloud`: Fedora Cloud container disk
- `cirros`: CirrOS test image

## Python Client Example

```python
import requests
import json

class KViRTOS2Client:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        
    def create_server(self, name, image_ref, flavor_ref, metadata=None):
        data = {
            "server": {
                "name": name,
                "imageRef": image_ref,
                "flavorRef": flavor_ref,
                "metadata": metadata or {}
            }
        }
        response = self.session.post(f"{self.base_url}/v2.1/servers", json=data)
        return response.json()
        
    def list_servers(self):
        response = self.session.get(f"{self.base_url}/v2.1/servers")
        return response.json()
        
    def get_server(self, server_id):
        response = self.session.get(f"{self.base_url}/v2.1/servers/{server_id}")
        return response.json()
        
    def delete_server(self, server_id):
        response = self.session.delete(f"{self.base_url}/v2.1/servers/{server_id}")
        return response.status_code == 204
        
    def start_server(self, server_id):
        data = {"os-start": None}
        response = self.session.post(f"{self.base_url}/v2.1/servers/{server_id}/action", json=data)
        return response.status_code == 202
        
    def stop_server(self, server_id):
        data = {"os-stop": None}
        response = self.session.post(f"{self.base_url}/v2.1/servers/{server_id}/action", json=data)
        return response.status_code == 202

# Usage example
client = KViRTOS2Client()

# Create a VM
vm = client.create_server("my-test-vm", "fedora-cloud", "m1.small", {"env": "test"})
vm_id = vm["server"]["id"]

# Start the VM
client.start_server(vm_id)

# List all VMs
vms = client.list_servers()
print(json.dumps(vms, indent=2))
```

## Makefile

```makefile
.PHONY: build run test clean docker-build deploy

# Build the application
build:
	go build -o bin/kvirtos2 .

# Run locally
run:
	go run main.go --kubeconfig ~/.kube/config

# Test the application
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t kvirtos2:latest .

# Deploy to Kubernetes
deploy:
	kubectl apply -f deploy.yaml

# Remove from Kubernetes
undeploy:
	kubectl delete -f deploy.yaml

# Port forward for local access
port-forward:
	kubectl port-forward svc/kvirtos2 8080:8080
```

## Environment Variables

The following environment variables can be used to configure kvirtos2:

- `KUBECONFIG`: Path to kubeconfig file (optional)
- `NAMESPACE`: Kubernetes namespace to operate in (default: "default")
- `PORT`: Port to listen on (default: "8080")

## Troubleshooting

### Common Issues

1. **Permission denied errors**: Ensure the service account has proper RBAC permissions
2. **Image pull errors**: Verify the container disk images are accessible
3. **Network connectivity**: Check that the pod network is properly configured
4. **Resource constraints**: Ensure the cluster has sufficient CPU/memory resources

### Debugging

```bash
# Check pod logs
kubectl logs -l app=kvirtos2

# Check KubeVirt resources
kubectl get vms
kubectl get vmis

# Check events
kubectl get events --sort-by='.lastTimestamp'
```