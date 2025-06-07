# kvirtos2 REST API Documentation

## Overview

kvirtos2 provides a clean, Kubernetes-native REST API for managing virtual machines using KubeVirt. This documentation covers all available endpoints, request/response formats, and usage examples.

## API Design Principles

- **Kubernetes-native**: Use VM names (not UUIDs) as primary identifiers
- **RESTful**: Clear HTTP verbs and resource-based URLs
- **Simple**: Minimal required fields, sensible defaults
- **Extensible**: Designed for future volume and network features
- **Clear errors**: Descriptive error messages with proper HTTP status codes

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Currently no authentication required. Future versions may add:
- API keys
- Kubernetes RBAC integration
- JWT tokens

## Common Data Types

### VM Status Values
- `stopped` - VM exists but not running
- `starting` - VM is being started
- `running` - VM is active and running
- `stopping` - VM is being stopped
- `error` - VM is in error state
- `unknown` - Status cannot be determined

### Error Response Format
All API errors return a consistent format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "VM name must be a valid DNS label",
    "details": {
      "field": "name",
      "value": "invalid-name-",
      "constraint": "must match ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
    }
  }
}
```

### Error Codes
| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Request validation failed |
| `NOT_FOUND` | Resource not found |
| `ALREADY_EXISTS` | Resource already exists |
| `CONFLICT` | Operation conflicts with current state |
| `INTERNAL_ERROR` | Internal server error |
| `TIMEOUT` | Operation timed out |

## API Endpoints

### 1. Health Check

**GET** `/health`

Returns API health status and system information.

**Response: 200 OK**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "kubevirt": "ready"
}
```

**Example:**
```bash
curl http://localhost:8080/health
```

---

### 2. Create VM

**POST** `/api/v1/vms`

Create a new virtual machine.

**Request Body:**
```json
{
  "name": "my-new-vm",
  "image": "ubuntu-24",
  "flavor": "m1.small",
  "metadata": {
    "environment": "test",
    "project": "demo"
  },
  "autoStart": false
}
```

**Required Fields:**
- `name` (string) - VM name (DNS-1123 compliant: lowercase, alphanumeric, hyphens)
- `image` (string) - Image reference (see Available Images)
- `flavor` (string) - Flavor reference (see Available Flavors)

**Optional Fields:**
- `metadata` (object) - Key-value labels for VM organization
- `autoStart` (boolean) - Start VM immediately after creation (default: false)

**Response: 201 Created**
```json
{
  "vm": {
    "name": "my-new-vm",
    "status": "stopped",
    "image": "ubuntu-24",
    "flavor": "m1.small",
    "created": "2024-01-15T10:30:00Z",
    "updated": "2024-01-15T10:30:00Z",
    "metadata": {
      "environment": "test",
      "project": "demo"
    },
    "networks": [
      {
        "name": "default",
        "type": "pod"
      }
    ],
    "resources": {
      "cpu": "1000m",
      "memory": "2Gi"
    }
  }
}
```

**Error Responses:**
- `400 Bad Request` - Invalid request body or validation error
- `409 Conflict` - VM with same name already exists
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server",
    "image": "ubuntu-24",
    "flavor": "m1.medium",
    "metadata": {"env": "production"},
    "autoStart": true
  }'
```

---

### 3. List VMs

**GET** `/api/v1/vms`

List all virtual machines with optional filtering.

**Query Parameters:**
- `status` (optional) - Filter by status: `running`, `stopped`, `starting`, `stopping`, `error`
- `label` (optional) - Filter by metadata label: `environment=prod`, `team=web`

**Response: 200 OK**
```json
{
  "vms": [
    {
      "name": "web-server-1",
      "status": "running",
      "image": "ubuntu-24",
      "flavor": "m1.small",
      "created": "2024-01-15T10:30:00Z",
      "updated": "2024-01-15T10:32:15Z",
      "metadata": {
        "environment": "production",
        "owner": "team-web"
      },
      "networks": [
        {
          "name": "default",
          "type": "pod",
          "addresses": [
            {
              "ip": "10.244.1.5",
              "type": "internal"
            }
          ]
        }
      ],
      "node": "worker-node-1",
      "resources": {
        "cpu": "1000m",
        "memory": "2Gi"
      }
    }
  ],
  "total": 1
}
```

**Examples:**
```bash
# List all VMs
curl http://localhost:8080/api/v1/vms

# Filter by status
curl http://localhost:8080/api/v1/vms?status=running

# Filter by label
curl http://localhost:8080/api/v1/vms?label=environment=prod

# Multiple filters
curl "http://localhost:8080/api/v1/vms?status=running&label=team=web"
```

---

### 4. Get VM Details

**GET** `/api/v1/vms/{name}`

Get detailed information about a specific virtual machine.

**Path Parameters:**
- `name` (string) - VM name

**Response: 200 OK**
```json
{
  "vm": {
    "name": "my-vm",
    "status": "running",
    "image": "ubuntu-24",
    "flavor": "m1.small",
    "created": "2024-01-15T10:30:00Z",
    "updated": "2024-01-15T10:32:15Z",
    "metadata": {
      "environment": "test"
    },
    "networks": [
      {
        "name": "default",
        "type": "pod",
        "addresses": [
          {
            "ip": "10.244.1.5",
            "type": "internal"
          }
        ]
      }
    ],
    "node": "worker-node-1",
    "resources": {
      "cpu": "1000m",
      "memory": "2Gi"
    },
    "conditions": [
      {
        "type": "Ready",
        "status": "True",
        "lastTransitionTime": "2024-01-15T10:32:15Z"
      }
    ]
  }
}
```

**Error Responses:**
- `404 Not Found` - VM not found
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl http://localhost:8080/api/v1/vms/web-server
```

---

### 5. Update VM

**PUT** `/api/v1/vms/{name}`

Update VM metadata and configuration. Note: Currently only metadata updates are supported.

**Path Parameters:**
- `name` (string) - VM name

**Request Body:**
```json
{
  "metadata": {
    "environment": "staging",
    "version": "1.2.0",
    "owner": "team-platform"
  }
}
```

**Response: 200 OK**
Returns the updated VM object (same format as Get VM Details).

**Error Responses:**
- `400 Bad Request` - Invalid request body
- `404 Not Found` - VM not found
- `409 Conflict` - VM running, cannot update certain fields
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X PUT http://localhost:8080/api/v1/vms/web-server \
  -H "Content-Type: application/json" \
  -d '{"metadata": {"version": "2.0.0", "updated-by": "admin"}}'
```

---

### 6. Delete VM

**DELETE** `/api/v1/vms/{name}`

Delete a virtual machine.

**Path Parameters:**
- `name` (string) - VM name

**Query Parameters:**
- `force` (optional, boolean) - Force delete even if running (default: false)

**Response: 204 No Content**

**Error Responses:**
- `404 Not Found` - VM not found
- `409 Conflict` - VM is running and force=false
- `500 Internal Server Error` - Server error

**Examples:**
```bash
# Delete stopped VM
curl -X DELETE http://localhost:8080/api/v1/vms/web-server

# Force delete running VM
curl -X DELETE http://localhost:8080/api/v1/vms/web-server?force=true
```

---

### 7. Start VM

**POST** `/api/v1/vms/{name}/start`

Start a virtual machine.

**Path Parameters:**
- `name` (string) - VM name

**Request Body (Optional):**
```json
{
  "timeout": "5m"
}
```

**Response: 202 Accepted**
```json
{
  "message": "VM start initiated",
  "vm": {
    "name": "my-vm",
    "status": "starting"
  }
}
```

**Error Responses:**
- `404 Not Found` - VM not found
- `409 Conflict` - VM already running
- `422 Unprocessable Entity` - VM in error state
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/vms/web-server/start
```

---

### 8. Stop VM

**POST** `/api/v1/vms/{name}/stop`

Stop a virtual machine.

**Path Parameters:**
- `name` (string) - VM name

**Request Body (Optional):**
```json
{
  "graceful": true,
  "timeout": "30s"
}
```

**Response: 202 Accepted**
```json
{
  "message": "VM stop initiated",
  "vm": {
    "name": "my-vm",
    "status": "stopping"
  }
}
```

**Error Responses:**
- `404 Not Found` - VM not found
- `409 Conflict` - VM already stopped
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/vms/web-server/stop
```

---

### 9. Restart VM

**POST** `/api/v1/vms/{name}/restart`

Restart a virtual machine (stop then start).

**Path Parameters:**
- `name` (string) - VM name

**Request Body (Optional):**
```json
{
  "graceful": true,
  "timeout": "60s"
}
```

**Response: 202 Accepted**
```json
{
  "message": "VM restart initiated",
  "vm": {
    "name": "my-vm", 
    "status": "stopping"
  }
}
```

**Error Responses:**
- `404 Not Found` - VM not found
- `500 Internal Server Error` - Server error

**Example:**
```bash
curl -X POST http://localhost:8080/api/v1/vms/web-server/restart
```

---

### 10. Get VM Console

**GET** `/api/v1/vms/{name}/console`

Get console access information for a VM. (Future feature - endpoint exists but not fully implemented)

**Path Parameters:**
- `name` (string) - VM name

**Response: 200 OK**
```json
{
  "type": "vnc",
  "url": "ws://localhost:8080/api/v1/vms/my-vm/console/vnc",
  "token": "console-token-123"
}
```

## Resource Definitions

### VM Object Structure

The core VM object contains the following fields:

```json
{
  "name": "string",              // DNS-1123 compliant name
  "status": "string",            // VM status (see status values above)
  "image": "string",             // Image reference
  "flavor": "string",            // Flavor reference  
  "created": "2024-01-15T10:30:00Z",  // ISO 8601 timestamp
  "updated": "2024-01-15T10:32:15Z",  // ISO 8601 timestamp
  "metadata": {                  // Key-value labels
    "environment": "production",
    "team": "platform"
  },
  "networks": [                  // Network attachments
    {
      "name": "default",
      "type": "pod",
      "addresses": [
        {
          "ip": "10.244.1.5",
          "type": "internal",
          "version": 4
        }
      ]
    }
  ],
  "node": "worker-node-1",       // Kubernetes node name
  "resources": {                 // Actual resource allocation
    "cpu": "1000m",              // CPU in millicores
    "memory": "2Gi"              // Memory in Kubernetes format
  },
  "conditions": [                // Kubernetes-style conditions
    {
      "type": "Ready",
      "status": "True",
      "lastTransitionTime": "2024-01-15T10:32:15Z"
    }
  ]
}
```

### Network Object

```json
{
  "name": "string",              // Network name
  "type": "string",              // Network type: "pod", "bridge", "sriov"
  "addresses": [                 // Assigned IP addresses
    {
      "ip": "10.244.1.5",
      "type": "internal",         // Address type: "internal", "external", "floating"
      "version": 4                // IP version: 4 or 6
    }
  ]
}
```

### Resource Object

```json
{
  "cpu": "1000m",                // CPU allocation in millicores
  "memory": "2Gi"                // Memory allocation in Kubernetes format
}
```

## Available Images

Current supported images:

| Image | Description | Use Case |
|-------|-------------|----------|
| `ubuntu-24` | Ubuntu 24.04 LTS | Production workloads |
| `ubuntu-20.04` | Ubuntu 20.04 LTS | Legacy applications |
| `fedora-cloud` | Fedora Cloud Image | Development and testing |
| `cirros` | CirrOS Test Image | Quick testing and validation |

## Available Flavors

Current supported flavors:

| Flavor | CPU | Memory | Use Case |
|--------|-----|--------|----------|
| `m1.tiny` | 0.5 core (500m) | 512MB | Minimal testing |
| `m1.small` | 1 core (1000m) | 2GB | Light workloads |
| `m1.medium` | 2 cores (2000m) | 4GB | Standard applications |
| `m1.large` | 4 cores (4000m) | 8GB | Resource-intensive apps |

## Complete Usage Examples

### Example 1: Basic VM Lifecycle

```bash
# 1. Create a VM
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "demo-vm",
    "image": "ubuntu-24",
    "flavor": "m1.small",
    "metadata": {"purpose": "demo"}
  }'

# 2. Start the VM
curl -X POST http://localhost:8080/api/v1/vms/demo-vm/start

# 3. Check status (wait for "running")
curl http://localhost:8080/api/v1/vms/demo-vm

# 4. Stop the VM
curl -X POST http://localhost:8080/api/v1/vms/demo-vm/stop

# 5. Delete the VM
curl -X DELETE http://localhost:8080/api/v1/vms/demo-vm
```

### Example 2: Batch Operations

```bash
# Create multiple VMs
for i in {1..3}; do
  curl -X POST http://localhost:8080/api/v1/vms \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"web-${i}\",
      \"image\": \"ubuntu-24\",
      \"flavor\": \"m1.small\",
      \"metadata\": {\"role\": \"web\", \"instance\": \"${i}\"}
    }"
done

# Start all web VMs
for i in {1..3}; do
  curl -X POST http://localhost:8080/api/v1/vms/web-${i}/start
done

# List running web VMs
curl "http://localhost:8080/api/v1/vms?status=running&label=role=web"
```

### Example 3: Metadata Management

```bash
# Create VM with initial metadata
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "app-server",
    "image": "ubuntu-24",
    "flavor": "m1.medium",
    "metadata": {
      "environment": "staging",
      "version": "1.0.0",
      "team": "backend"
    }
  }'

# Update metadata (e.g., after deployment)
curl -X PUT http://localhost:8080/api/v1/vms/app-server \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "environment": "production",
      "version": "1.1.0",
      "team": "backend",
      "deployed-by": "ci-cd",
      "deployment-date": "2024-01-15"
    }
  }'

# Filter by updated metadata
curl "http://localhost:8080/api/v1/vms?label=environment=production"
```

### Example 4: Error Handling

```bash
# Try to create VM with invalid name
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{"name": "Invalid_Name!", "image": "ubuntu-24", "flavor": "m1.small"}'

# Expected response: 400 Bad Request
# {
#   "error": {
#     "code": "VALIDATION_ERROR",
#     "message": "VM name must be a valid DNS label",
#     "details": {
#       "field": "name",
#       "value": "Invalid_Name!",
#       "constraint": "must match ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
#     }
#   }
# }

# Try to start non-existent VM
curl -X POST http://localhost:8080/api/v1/vms/does-not-exist/start

# Expected response: 404 Not Found
# {
#   "error": {
#     "code": "NOT_FOUND",
#     "message": "VM 'does-not-exist' not found"
#   }
# }
```

## HTTP Status Codes

### Success Codes
- `200 OK` - Request successful, data returned
- `201 Created` - Resource created successfully
- `202 Accepted` - Request accepted, processing asynchronously
- `204 No Content` - Request successful, no data returned

### Client Error Codes
- `400 Bad Request` - Invalid request format or parameters
- `404 Not Found` - Resource not found
- `409 Conflict` - Request conflicts with current state
- `422 Unprocessable Entity` - Request valid but cannot be processed

### Server Error Codes
- `500 Internal Server Error` - Server error occurred

## Rate Limiting

Currently no rate limiting is implemented. Future versions may include:
- Request rate limiting per client IP
- Concurrent operation limits per VM
- Resource usage quotas

## Versioning

The API uses URL-based versioning (`/api/v1/`). Future versions will:
- Maintain backward compatibility within major versions
- Provide migration guides for major version changes
- Support parallel versions during transition periods

## WebSocket Support (Future)

Planned WebSocket endpoints for real-time features:
- `/api/v1/vms/{name}/console/vnc` - VNC console access
- `/api/v1/vms/{name}/console/serial` - Serial console access
- `/api/v1/events` - Real-time VM status events

## SDK and Client Libraries

The clean REST API design makes it suitable for generating client libraries in various languages:
- Go SDK (planned)
- Python SDK (planned)
- JavaScript/TypeScript SDK (planned)
- OpenAPI/Swagger specification (planned)

---

For implementation details and deployment instructions, see the main [README.md](README.md) file.