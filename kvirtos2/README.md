# kvirtos2

The goal of kvirtos2 is to provide a lightweight, easy-to-use, and flexible virtualization solution for Kubernetes clusters. It is essentially a REST API that allows users to create and manage virtual machines (VMs) on their Kubernetes clusters using a simple and intuitive interface.

VMs are created using a REST API that is compatible with OpenStack Nova, and in the backend translates those requests into Kubernetes resources using KubeVirt.

## Features

The implementation contains the following features:

- ✅ Create VMs
- ✅ Delete VMs
- ✅ List VMs
- ✅ Start VMs
- ✅ Stop VMs
- ✅ Get VM details
- ✅ Reboot VMs
- ✅ OpenStack Nova API compatibility
- ✅ Default pod networking
- ✅ Metadata support

When creating a VM it always gets a default network like this in KubeVirt:

```yaml
networks:
  - name: default
    pod: {}
```

Extra networks can be implemented in a later phase.

## Architecture

kvirtos2 is built using:

- **Go** with the Gin web framework
- **Kubernetes API clients** for cluster interaction
- **KubeVirt API clients** for VM management
- **OpenStack Nova API compatibility** for familiar interface

## Quick Start

### Prerequisites

- Kubernetes cluster with KubeVirt installed
- `kubectl` configured to access the cluster
- Go 1.21+ (for local development)
- Docker (for containerized deployment)

### Local Development

```bash
# Clone and setup
git clone <repository>
cd kvirtos2

# Download dependencies
go mod tidy

# Run locally (requires kubeconfig)
go run main.go --kubeconfig ~/.kube/config --namespace default --port 8080
```

### Docker Deployment

```bash
# Build Docker image
make docker-build

# Deploy to Kubernetes
make deploy

# Port forward for local access
make port-forward
```

## API Usage

### Create a VM

```bash
curl -X POST http://localhost:8080/v2.1/servers \
  -H "Content-Type: application/json" \
  -d '{
    "server": {
      "name": "test-vm",
      "imageRef": "fedora-cloud",
      "flavorRef": "m1.small",
      "metadata": {
        "purpose": "testing"
      }
    }
  }'
```

### List VMs

```bash
curl http://localhost:8080/v2.1/servers
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

### Delete a VM

```bash
curl -X DELETE http://localhost:8080/v2.1/servers/{vm-id}
```

## Available Resources

### Flavors

- `m1.tiny`: 0.5 CPU, 512MB RAM
- `m1.small`: 1 CPU, 2GB RAM  
- `m1.medium`: 2 CPU, 4GB RAM
- `m1.large`: 4 CPU, 8GB RAM

### Images

- `ubuntu-20.04`: Ubuntu 20.04 container disk
- `fedora-cloud`: Fedora Cloud container disk
- `cirros`: CirrOS test image

## Configuration

Environment variables:

- `KUBECONFIG`: Path to kubeconfig file (optional, uses in-cluster config if not provided)
- `NAMESPACE`: Kubernetes namespace to operate in (default: "default")
- `PORT`: Port to listen on (default: "8080")

Command line flags:

```bash
./kvirtos2 --kubeconfig /path/to/config --namespace default --port 8080
```

## Development

### Project Structure

```
kvirtos2/
├── main.go                 # Application entry point
├── internal/
│   ├── handlers/           # HTTP request handlers
│   ├── kubevirt/          # KubeVirt client wrapper
│   └── models/            # Data models
├── deploy.yaml            # Kubernetes deployment manifests
├── Dockerfile             # Container build definition
├── Makefile              # Build and deployment tasks
└── examples.md           # Usage examples
```

### Building

```bash
# Local binary
make build

# Docker image
make docker-build

# Deploy to cluster
make deploy
```

### Testing

```bash
# Run tests
make test

# Check deployment
make status

# View logs
make logs
```

## OpenStack Nova Compatibility

kvirtos2 implements a subset of the OpenStack Nova API for maximum compatibility:

- `POST /v2.1/servers` - Create server
- `GET /v2.1/servers` - List servers  
- `GET /v2.1/servers/{id}` - Show server details
- `DELETE /v2.1/servers/{id}` - Delete server
- `POST /v2.1/servers/{id}/action` - Server actions (start, stop, reboot)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0.
