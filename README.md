# Complete Guide: Talos Cluster with Docker, KubeVirt, Multus, and VM Setup

This comprehensive guide covers everything you need to set up a Talos OS cluster using Docker, install KubeVirt and Multus CNI, and deploy virtual machines with multiple network interfaces. Multus CNI will handle the bridge creation automatically through network attachment definitions.

We will also add 2 vxlans, that will allow our multi-tenant virtual machines to communicate with each other. Even when they have overlapping address ranges.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Step 1: Install Talos CLI Tool](#step-1-install-talos-cli-tool)
- [Step 2: Create a Talos Cluster with Docker](#step-2-create-a-talos-cluster-with-docker)
- [Step 3: Install KubeVirt](#step-3-install-kubevirt)
- [Step 4: Install Multus CNI](#step-4-install-multus-cni)
- [Step 5: Configure Network Attachments](#step-5-configure-network-attachments)
- [Step 6: Create and Deploy VMs](#step-6-create-and-deploy-vms)
- [Step 7: VXLAN Setup](#step-7-vxlan-setup)
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

We've created a shell script that handles all the KubeVirt installation steps automatically. This script:
- Creates the necessary namespace
- Downloads and applies the KubeVirt operator
- Creates the KubeVirt custom resource
- Waits for KubeVirt to become ready
- Installs the virtctl tool for VM management

Run the installation script:

```bash
# Make the script executable (if not already)
chmod +x install-kubevirt.sh

# Run the installation script
./install-kubevirt.sh
```

Once the script completes successfully, KubeVirt will be installed and ready to use.

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

We'll create two different NetworkAttachmentDefinitions to demonstrate the Whereabouts IPAM plugin, and show overlapping IP ranges.

To deploy the network attachment definitions, run the following commands:

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

We will create 2 virtual machines, this is done by applying the following yaml files:

```bash
kubectl apply -f multinet-vm1.yaml
kubectl apply -f multinet-vm2.yaml
kubectl apply -f multinet-vm3.yaml
kubectl apply -f multinet-vm4.yaml
```

Monitor VM creation:

```bash
kubectl get vms
kubectl get vmis
```

Start the console to all VMs like this:

```bash
kubectl virt console vm1 # user: ubuntu password: password
kubectl virt console vm2 # user: ubuntu password: password
kubectl virt console vm3 # user: ubuntu password: password
kubectl virt console vm4 # user: ubuntu password: password
```

and run `sudo dhclient`.

## Step 7: VXLAN Setup

Deploy the `netadmin-ds.yaml` tooling:

```bash
kubectl apply -f netadmin-ds.yaml
```

Now exec into the pods on each host:

```bash
kubectl exec -it netadmin-ds-<pod-id> -n kube-system -- /bin/bash
```

and run the following commands:

```bash
ip link add vxlan100 type vxlan id 100 group 239.1.1.1 dev eth0 dstport 4789 ; ip link set vxlan100 up ; ip link set vxlan100 master br1
ip link add vxlan200 type vxlan id 200 group 239.1.1.1 dev eth0 dstport 4789 ; ip link set vxlan200 up ; ip link set vxlan200 master br2
```

### Tenant Configuration

The following table shows the multi-tenant setup with VMs and their network configurations:

| Tenant | VMs | Subnet | VXLAN ID | Bridge | Node Placement |
|--------|-----|--------|----------|--------|----------------|
| Tenant 1 | vm1, vm3 | 192.168.1.0/24 | vxlan100 | br1 | Anti-affinity between VMs |
| Tenant 2 | vm2, vm4 | 192.168.1.0/24 | vxlan200 | br2 | Anti-affinity between VMs |

This configuration demonstrates how multiple tenants can use the same IP address range (192.168.1.0/24) without conflicts by isolating their traffic through separate VXLAN networks. The VMs are placed on different nodes using anti-affinity rules to ensure high availability.


## Troubleshooting

???

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
