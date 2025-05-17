# Complete Guide: Talos Cluster with Docker, KubeVirt, Multus, and VM Setup

This comprehensive guide covers everything you need to set up a Talos OS cluster using Docker, install KubeVirt and Multus CNI, and deploy virtual machines with multiple network interfaces. Multus CNI will handle the bridge creation automatically through network attachment definitions.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Step 1: Install Talos CLI Tool](#step-1-install-talos-cli-tool)
- [Step 2: Create a Talos Cluster with Docker](#step-2-create-a-talos-cluster-with-docker)
- [Step 3: Install KubeVirt](#step-3-install-kubevirt)
- [Step 4: Install Multus CNI](#step-4-install-multus-cni)
- [Step 5: Configure Network Attachments](#step-5-configure-network-attachments)
- [Step 6: Create and Deploy VMs](#step-6-create-and-deploy-vms)
- [Troubleshooting](#troubleshooting)
- [Cleanup](#cleanup)

## Prerequisites

- Linux machine with Docker installed
- At least 8GB RAM and 4 CPU cores available
- `kubectl` installed
- Root or sudo access
- Internet connection for downloading components

## Step 1: Install Talos CLI Tool

Download and install the Talos CLI tool (talosctl):

```bash
# Download the latest talosctl binary
curl -Lo /tmp/talosctl https://github.com/siderolabs/talos/releases/latest/download/talosctl-linux-amd64
chmod +x /tmp/talosctl
sudo mv /tmp/talosctl /usr/local/bin/
```

Verify installation:

```bash
talosctl version
```

## Step 2: Create a Talos Cluster with Docker

Create a multi-node Talos cluster:

```bash
# Create a cluster with 1 control plane and 2 worker nodes
talosctl cluster create --workers 2
```

This command:
- Creates Docker containers as virtual Talos nodes
- Sets up a Kubernetes control plane
- Configures worker nodes
- Creates a load balancer for the API server
- Generates and installs cluster certificates

Once completed, verify your cluster is running:

```bash
# Check node status
kubectl get nodes

# View cluster information
kubectl cluster-info
```

The cluster configuration is stored in `~/.talos/config`. To interact with your nodes:

```bash
# List nodes in the cluster
talosctl --nodes 10.5.0.2 get members
```

Replace `10.5.0.2` with the actual IP of your control plane node.

## Step 3: Install KubeVirt

1. Create the necessary namespaces:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: kubevirt
---
apiVersion: v1
kind: Namespace
metadata:
  name: cdi
EOF
```

2. Deploy the KubeVirt operator:

```bash
# Get the latest KubeVirt version
export VERSION=$(curl -s https://api.github.com/repos/kubevirt/kubevirt/releases | grep tag_name | grep -v -- '-rc' | sort -r | head -1 | awk -F': ' '{print $2}' | sed 's/,//' | xargs)

# Deploy the KubeVirt operator
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml
```

3. Create the KubeVirt custom resource:

```bash
kubectl apply -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
```

4. Wait for KubeVirt to become ready:

```bash
kubectl -n kubevirt wait kv kubevirt --for condition=Available --timeout=300s
```

5. Install the virtctl tool for VM management:

```bash
curl -L -o virtctl https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/virtctl-${VERSION}-linux-amd64
chmod +x virtctl
sudo mv virtctl /usr/local/bin
```

## Step 4: Install Multus CNI and Whereabouts IPAM

Install the Multus CNI for additional network interfaces:

```bash
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml
```

Verify Multus installation:

```bash
kubectl get pods -n kube-system | grep multus
```

Now install the Whereabouts IPAM plugin for dynamic IP address management:

```bash
# Install the Whereabouts IPAM plugin
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/daemonset-install.yaml

# Install the Whereabouts CRDs
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_ippools.yaml
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml
```

Verify Whereabouts installation:

```bash
kubectl get pods -n kube-system | grep whereabouts
```

## Step 5: Configure Network Attachments

We'll create two different NetworkAttachmentDefinitions to demonstrate both host-local and Whereabouts IPAM plugins.

First, create a file named `bridge-network.yaml` for the bridge network using host-local IPAM:

```bash
kubectl apply -f bridge-network.yaml
kubectl apply -f bridge-network2.yaml
```

Verify the network attachment definition:

```bash
kubectl get network-attachment-definitions
```

When these NetworkAttachmentDefinitions are applied, Multus CNI will automatically create and configure the bridge interfaces (`br1` and `br2`) on each node in the cluster. Unlike the manual approach, this method leverages Kubernetes' native networking capabilities, letting Multus handle the bridge creation and management automatically when VMs are scheduled.


## Step 6: Create and Deploy VMs

Create a VM with multiple network interfaces:

```bash
kubectl apply -f multinet-vm1.yaml
kubectl apply -f multinet-vm2.yaml
```

Monitor VM creation:

```bash
kubectl get vms
kubectl get vmis
```

Once the VM is running, access the console:

```bash
virtctl console multinet-vm
```

Inside the VM, verify the network interfaces:

```bash
# Inside the VM
ip a

# Test connectivity
ping -c 3 8.8.8.8        # Default pod network
ping -c 3 192.168.100.1  # Bridge network
```

## Troubleshooting

### KubeVirt Issues

1. Check KubeVirt pod status:
```bash
kubectl get pods -n kubevirt
```

2. View KubeVirt operator logs:
```bash
kubectl logs -n kubevirt -l kubevirt.io=virt-operator
```

3. Check KubeVirt custom resource status:
```bash
kubectl get kv -n kubevirt kubevirt -o yaml
```

### Network Issues

1. Verify Multus and Whereabouts are running:
```bash
kubectl get pods -n kube-system | grep -E 'multus|whereabouts'
```

2. Check NetworkAttachmentDefinition:
```bash
kubectl get network-attachment-definitions -o yaml
```

3. For VMs with network issues, check if the Multus annotation is correct:
```bash
kubectl get vmi vm1 -o yaml | grep -A 5 annotations
```

4. Check the IP addresses assigned to VM interfaces:
```bash
kubectl get vmi vm1 -o jsonpath='{range .status.interfaces[*]}{.name}{": "}{.ipAddress}{"\n"}{end}'
```

5. Check if Whereabouts IP reservations are correctly created:
```bash
kubectl get ippools.whereabouts.cni.cncf.io -A
```

### VM Issues

1. Check VM status:
```bash
kubectl get vmi
kubectl describe vmi multinet-vm
```

2. View VM logs:
```bash
# Replace with your pod name
kubectl logs virt-launcher-vm1-xxxxx
```

3. Force restart a VM:
```bash
virtctl restart vm1
```

4. Check multiple interfaces inside the VM (once you connect to VM console):
```bash
# Inside the VM
ip addr show
```

## Cleanup

To delete the VMs:

```bash
kubectl delete vm vm1
```

To remove KubeVirt:

```bash
kubectl delete -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
kubectl delete -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml
```

To remove Multus and Whereabouts:

```bash
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/daemonset-install.yaml
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_ippools.yaml
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml
```

To destroy the Talos cluster:

```bash
talosctl cluster destroy
```
