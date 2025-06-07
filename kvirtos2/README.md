# kvirtos2

A lightweight, Kubernetes-native REST API for managing virtual machines using KubeVirt. kvirtos2 provides a simple and intuitive interface for VM lifecycle management while leveraging native Kubernetes resources.

## Overview

kvirtos2 eliminates the complexity of traditional cloud APIs by providing a clean, RESTful interface for VM management on Kubernetes clusters. It translates simple HTTP requests into native Kubernetes resources using KubeVirt, making VM management as easy as managing any other Kubernetes workload.

## Features

- ✅ **VM Lifecycle Management**: Create, start, stop, restart, and delete VMs
- ✅ **Kubernetes-Native**: Uses VM names (not UUIDs) as identifiers
- ✅ **RESTful API**: Clean HTTP verbs and resource-based URLs
- ✅ **Metadata Support**: Key-value labels for VM organization
- ✅ **Resource Management**: Predefined flavors with CPU/memory allocation
- ✅ **Multiple Images**: Support for Ubuntu, Fedora, and CirrOS images
- ✅ **Default Networking**: Automatic pod network configuration
- ✅ **Health Monitoring**: Built-in health check endpoints
- ✅ **Container Ready**: Docker image with Kubernetes deployment manifests

## Architecture

### Technology Stack
- **Language**: Go 1.21+
- **Web Framework**: Gin HTTP framework
- **Kubernetes Integration**: Dynamic client (k8s.io/client-go)
- **Container Runtime**: Docker with multi-stage builds
- **API Design**: Clean v1 REST API (no legacy compatibility)

### Project Structure
```
kvirtos2/
├── main.go                    # Application entry point
├── go.mod                     # Go module dependencies
├── internal/
│   ├── handlers/              # HTTP request handlers
│   │   └── v1.go             # V1 API endpoints
│   ├── kubevirt/             # KubeVirt client components
│   │   ├── client.go         # Main orchestration client
│   │   ├── vm_builder.go     # VM specification builder
│   │   ├── status_mapper.go  # VM status mapping
│   │   ├── config.go         # Flavors and images
│   │   └── *.go              # Other focused components
│   └── models/               # Data models
│       ├── v1.go             # V1 API models
│       └── nova.go           # Legacy models (compatibility)
├── Dockerfile                # Multi-stage container build
├── deploy.yaml              # Kubernetes deployment manifests
└── Makefile                 # Build and deployment automation
```

## Quick Start

### Prerequisites

- Kubernetes cluster with KubeVirt installed and configured
- `kubectl` configured to access your cluster
- Go 1.21+ (for local development)
- Docker (for containerized deployment)

### Local Development

```bash
# Clone the repository
git clone <repository-url>
cd kvirtos2

# Install dependencies
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

### Quick Test

```bash
# Check API health
curl http://localhost:8080/health

# Create a VM
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-vm",
    "image": "ubuntu-24",
    "flavor": "m1.small",
    "autoStart": true
  }'

# List VMs
curl http://localhost:8080/api/v1/vms

# Check VM status
curl http://localhost:8080/api/v1/vms/test-vm
```

## API Overview

### Base URL
```
http://localhost:8080/api/v1
```

### Core Endpoints
- `GET /health` - API health check
- `POST /api/v1/vms` - Create VM
- `GET /api/v1/vms` - List VMs
- `GET /api/v1/vms/{name}` - Get VM details
- `PUT /api/v1/vms/{name}` - Update VM metadata
- `DELETE /api/v1/vms/{name}` - Delete VM
- `POST /api/v1/vms/{name}/start` - Start VM
- `POST /api/v1/vms/{name}/stop` - Stop VM
- `POST /api/v1/vms/{name}/restart` - Restart VM

### Example: Create and Manage VM

```bash
# Create VM with metadata
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server",
    "image": "ubuntu-24",
    "flavor": "m1.medium",
    "metadata": {
      "environment": "production",
      "team": "platform"
    },
    "autoStart": false
  }'

# Start the VM
curl -X POST http://localhost:8080/api/v1/vms/web-server/start

# Check status
curl http://localhost:8080/api/v1/vms/web-server

# Update metadata
curl -X PUT http://localhost:8080/api/v1/vms/web-server \
  -H "Content-Type: application/json" \
  -d '{"metadata": {"version": "1.2.0"}}'

# Stop and delete
curl -X POST http://localhost:8080/api/v1/vms/web-server/stop
curl -X DELETE http://localhost:8080/api/v1/vms/web-server
```

## Available Resources

### Flavors (CPU/Memory)
- `m1.tiny`: 0.5 CPU, 512MB RAM
- `m1.small`: 1 CPU, 2GB RAM
- `m1.medium`: 2 CPU, 4GB RAM
- `m1.large`: 4 CPU, 8GB RAM

### Images
- `ubuntu-24`: Ubuntu 24.04 LTS
- `ubuntu-20.04`: Ubuntu 20.04 LTS
- `fedora-cloud`: Fedora Cloud Image
- `cirros`: CirrOS Test Image

## Configuration

### Environment Variables
- `KUBECONFIG`: Path to kubeconfig file (optional, uses in-cluster config if not provided)
- `NAMESPACE`: Kubernetes namespace to operate in (default: "default")
- `PORT`: Port to listen on (default: "8080")

### Command Line Flags
```bash
./kvirtos2 --kubeconfig /path/to/config --namespace kube-system --port 8080
```

## Deployment

### Kubernetes Deployment

The included `deploy.yaml` provides:
- Deployment with proper resource limits
- Service for cluster access
- ServiceAccount with minimal RBAC permissions
- ConfigMap for configuration

```bash
# Deploy to cluster
kubectl apply -f deploy.yaml

# Check deployment
kubectl get pods -l app=kvirtos2

# Access via port-forward
kubectl port-forward svc/kvirtos2 8080:8080
```

### Docker Image

Multi-stage Dockerfile optimized for:
- Minimal runtime image size
- Security (non-root user)
- Efficient layer caching
- Static binary compilation

## Networking

All VMs automatically receive default pod networking:
```yaml
networks:
  - name: default
    pod: {}
```

This provides:
- Cluster-internal IP address
- Access to cluster services
- Network policies compatibility
- Simple, predictable networking

## Development

### Building

```bash
# Local binary
make build

# Docker image
make docker-build

# Run tests
make test
```

### Testing

The project includes comprehensive test coverage:
- Model validation tests
- Handler unit tests
- Integration test examples
- Mock client for testing

```bash
# Run all tests
go test ./...

# Run specific test suites
go test ./internal/models/...
go test ./internal/handlers/...
```

### Code Organization

The codebase follows clean architecture principles:
- **Handlers**: HTTP request/response handling
- **Models**: Data structures and validation
- **KubeVirt Client**: Kubernetes resource management
- **Interfaces**: Testable contracts between components

## Security & RBAC

The deployment includes minimal required permissions:
- VirtualMachine resource management
- VirtualMachineInstance read access
- Event and status updates
- No cluster-admin privileges required

## Monitoring

Built-in observability features:
- Health check endpoint (`/health`)
- Structured JSON logging
- HTTP request logging
- Error tracking and reporting

## Future Roadmap

### Phase 2: Enhanced Features
- Multiple network attachments
- Volume management (PVCs, ConfigMaps)
- VM console access
- Resource quotas and limits

### Phase 3: Advanced Operations  
- VM migration support
- Backup and restore
- Advanced networking (SR-IOV, bridge)
- Multi-tenancy support

### Phase 4: Enterprise Features
- Authentication and authorization
- API rate limiting
- Metrics and monitoring integration
- Webhook support for external integrations

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines
- Follow Go best practices and formatting
- Add tests for new functionality
- Update documentation for API changes
- Ensure Docker builds succeed
- Test with a real KubeVirt cluster

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Support

- **Documentation**: See `api-docs.md` for complete API reference
- **Issues**: Report bugs and feature requests via GitHub Issues
- **Community**: Join discussions in the project repository

---

**kvirtos2** - Simple VM management for Kubernetes clusters