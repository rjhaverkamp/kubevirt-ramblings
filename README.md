# Complete Guide: Talos Cluster with Docker, KubeVirt, Multus, and VM Setup

This comprehensive guide covers everything you need to set up a Talos OS cluster using Docker, install KubeVirt and Multus CNI, and deploy virtual machines with multiple network interfaces. Multus CNI will handle the bridge creation automatically through network attachment definitions.

We will also add VXLANs that will allow our multi-tenant virtual machines to communicate with each other. Even when they have overlapping address ranges.

The project files are organized into directories:
- `evpn/` - BGP EVPN routing configurations
- `nets/` - Network attachment definitions (bridge-network10 through bridge-network20)
- `vm/` - Virtual machine definitions (multinet-vm1 through multinet-vm6)
- `install-kubevirt.sh` - Installation script
- `Ubuntuimage` - Container image definition

## Table of Contents
- [Prerequisites](#prerequisites)
- [Step 1: Install Talos CLI Tool](#step-1-install-talos-cli-tool)
- [Step 2: Create a Talos Cluster with Docker](#step-2-create-a-talos-cluster-with-docker)
- [Step 3: Install KubeVirt, Multus CNI, and Whereabouts IPAM](#step-3-install-kubevirt-multus-cni-and-whereabouts-ipam)
- [Step 4: Configure Network Attachments](#step-4-configure-network-attachments)
- [Step 5: Create and Deploy VMs](#step-5-create-and-deploy-vms)
- [Step 6: VXLAN Setup](#step-6-vxlan-setup)
- [Step 7: Advanced VXLAN over EVPN with BGP](#step-7-advanced-vxlan-over-evpn-with-bgp)
- [Troubleshooting](#troubleshooting)
- [Cleanup](#cleanup)

## Prerequisites

- Linux machine with Docker installed
- At least 8GB RAM and 4 CPU cores available
- `kubectl` installed
- Root or sudo access
- Internet connection for downloading components

## Step 1: Install Talos CLI Tool and krew setup

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


Install the Krew plugin manager:

```
(
  set -x; cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  KREW="krew-${OS}_${ARCH}" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
  tar zxvf "${KREW}.tar.gz" &&
  ./"${KREW}" install krew
)
export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH" # add this line to your shell profile to make it permanent
```

now install the `kubevirt` kubectl plugin:

```
kubectl krew install kubevirt
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

### trouble shooting talos cluster creation

When you're running this guide on linux and run into issues during cluster creation, related to kube-dns/core dns you might need to run these commands:

```
echo "net.bridge.bridge-nf-call-iptables = 1" >> /etc/sysctl.conf
modprobe bridge
modprobe br_netfilter
sysctl -p /etc/sysctl.conf
```

## Step 3: Install KubeVirt, Multus CNI, and Whereabouts IPAM

We've created a shell script that handles all the installation steps automatically. This script:
- Creates the necessary namespace
- Downloads and applies the KubeVirt operator
- Creates the KubeVirt custom resource
- Waits for KubeVirt to become ready
- Installs the virtctl tool for VM management
- Installs and configures Multus CNI for additional network interfaces
- Installs Whereabouts IPAM plugin for dynamic IP address management

Run the installation script:

```bash
# Make the script executable (if not already)
chmod +x install-kubevirt.sh

# Run the installation script
./install-kubevirt.sh
```

Once the script completes successfully, KubeVirt, Multus CNI, and Whereabouts IPAM will be installed and ready to use.

You can verify the installations with:

```bash
# Verify KubeVirt
kubectl get pods -n kubevirt

# Verify Multus
kubectl get pods -n kube-system | grep multus

# Verify Whereabouts
kubectl get pods -n kube-system | grep whereabouts
```

## Step 4: Configure Network Attachments

We'll create two different NetworkAttachmentDefinitions to demonstrate the Whereabouts IPAM plugin, and show overlapping IP ranges.

To deploy the network attachment definitions, run the following commands:

```bash
kubectl apply -f nets/bridge-network10.yaml
kubectl apply -f nets/bridge-network11.yaml
```

**Note:** The current VM configurations reference `bridge-network1`, `bridge-network2`, etc., but the actual network files are named `bridge-network10.yaml`, `bridge-network11.yaml`, etc. You may need to update the VM files to match the correct network names, or create the missing bridge network files.

Verify the network attachment definition:

```bash
kubectl get network-attachment-definitions
```

When these NetworkAttachmentDefinitions are applied, Multus CNI will automatically create and configure the bridge interfaces (`br1` and `br2`) on each node in the cluster. Unlike the manual approach, this method leverages Kubernetes' native networking capabilities, letting Multus handle the bridge creation and management automatically when VMs are scheduled.


## Step 5: Create and Deploy VMs

We will create virtual machines, this is done by applying the following yaml files:

```bash
kubectl apply -f vm/multinet-vm1.yaml
kubectl apply -f vm/multinet-vm2.yaml
kubectl apply -f vm/multinet-vm3.yaml
kubectl apply -f vm/multinet-vm4.yaml
```

**Note:** Make sure the network attachment definitions referenced in the VM files exist. The VMs currently reference networks like `bridge-network1`, `bridge-network2`, etc.

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

## Step 6: VXLAN Setup

Deploy the networking namespace and tooling:

```bash
kubectl apply -f evpn/namespace.yaml
kubectl apply -f evpn/setup-vxlan-bridges-configmap.yaml
```

Now exec into the networking pods on each host (after deploying the EVPN infrastructure):

```bash
kubectl exec -it <networking-pod-name> -n networking -- /bin/bash
```

and run the following commands:

```bash
ip link add vxlan10 type vxlan id 10 group 239.1.1.1 dev eth0 dstport 4789 ; ip link set vxlan10 up ; ip link set vxlan10 master br10
ip link add vxlan11 type vxlan id 11 group 239.1.1.1 dev eth0 dstport 4789 ; ip link set vxlan11 up ; ip link set vxlan11 master br11
```

### Tenant Configuration

The following table shows the multi-tenant setup with VMs and their network configurations:

| Tenant | VMs | Subnet | VXLAN ID | Bridge | Node Placement |
|--------|-----|--------|----------|--------|----------------|
| Tenant 1 | vm1, vm3 | 192.168.10.0/24 | vxlan10 | br10 | Anti-affinity between VMs |
| Tenant 2 | vm2, vm4 | 192.168.11.0/24 | vxlan11 | br11 | Anti-affinity between VMs |

This configuration demonstrates how multiple tenants can use overlapping IP address ranges without conflicts by isolating their traffic through separate VXLAN networks. The available network configurations include bridge networks 10-20, each with their own subnet ranges. The VMs are placed on different nodes using anti-affinity rules to ensure high availability.

## Step 7: Advanced VXLAN over EVPN with BGP

For more advanced networking scenarios, we can deploy BGP EVPN routers on each worker node to enable EVPN (Ethernet VPN) with VXLAN overlays. This provides dynamic MAC and IP learning across the cluster.

### Deploy BGP EVPN Routers

First, deploy the BGP EVPN configurations on both worker nodes:

```bash
kubectl apply -f evpn/evpn-control-1-ds.yaml
kubectl apply -f evpn/evpn-worker-1-ds.yaml
kubectl apply -f evpn/evpn-worker-2-ds.yaml
```

These deployments will:
- Install FRR on the control plane node
- Install FRR (Free Range Routing) on each worker node
- The bgp setup uses ebgp, the control plane doesn't announce anything but thanks to ebgp acts as a route refelctor.
- Configure BGP with EVPN address family
- Create VXLAN 300 with automatic bridge attachment

### Create Advanced Network Attachment

Deploy additional bridge networks as needed (networks 10-20 are available):

```bash
# Deploy additional networks from the nets/ directory
kubectl apply -f nets/bridge-network12.yaml
kubectl apply -f nets/bridge-network13.yaml
kubectl apply -f nets/bridge-network14.yaml
# ... apply other networks as needed (up to bridge-network20.yaml)
```

These create NetworkAttachmentDefinitions with different subnet ranges. Each network uses its own bridge (br10, br11, br12, etc.) and subnet (192.168.10.0/24, 192.168.11.0/24, etc.).

### Deploy VMs with EVPN Connectivity

Create VMs that will use the EVPN-enabled network:

```bash
kubectl apply -f vm/multinet-vm5.yaml
kubectl apply -f vm/multinet-vm6.yaml
```

### Verify EVPN Configuration

Once deployed, you can verify the BGP EVPN setup:

```bash
# Check that EVPN pods are running
kubectl get pods -n networking

# Exec into an EVPN pod to check BGP status
kubectl exec -it ubuntu-evpn-worker-1-<pod-id> -n networking -- vtysh -c "show bgp l2vpn evpn summary"

# Check VXLAN interface and bridge configuration
kubectl exec -it ubuntu-evpn-worker-1-<pod-id> -n networking -- ip link show
kubectl exec -it ubuntu-evpn-worker-1-<pod-id> -n networking -- brctl show
```

### Testing EVPN Connectivity

Start the VMs and test connectivity:

```bash
# Start VM consoles
kubectl virt console vm5  # user: ubuntu password: password
kubectl virt console vm6  # user: ubuntu password: password
```

Inside each VM, configure the bridge interface and test connectivity:

```bash
# In both VMs, configure the bridge interface
sudo dhclient

# Check IP assignment
ip a

# Test connectivity between VMs (they should be able to ping each other)
ping <other-vm-ip>
```

The EVPN setup provides:
- Automatic MAC address learning and distribution
- Dynamic VXLAN tunnel establishment
- Layer 2 connectivity across nodes with Layer 3 underlay
- BGP-based control plane for network advertisements

## File Structure

The project is now organized into the following directories:

- `evpn/` - Contains EVPN and BGP routing configurations
  - `namespace.yaml` - Networking namespace for BGP/EVPN pods
  - `evpn-control-1-ds.yaml` - Control plane BGP router
  - `evpn-worker-1-ds.yaml` - Worker 1 BGP router
  - `evpn-worker-2-ds.yaml` - Worker 2 BGP router
  - `setup-vxlan-bridges-configmap.yaml` - VXLAN bridge setup configuration
- `nets/` - Network attachment definitions
  - `bridge-network10.yaml` through `bridge-network20.yaml` - Various bridge networks with different subnets
- `vm/` - Virtual machine definitions
  - `multinet-vm1.yaml` through `multinet-vm6.yaml` - VM configurations with multiple network interfaces
- `install-kubevirt.sh` - Installation script for KubeVirt, Multus, and Whereabouts
- `Ubuntuimage` - Container image definition for Ubuntu VMs

## Troubleshooting

### Network Naming Mismatch

If you encounter issues with VMs not starting or network interfaces not being created, check for network naming mismatches:

1. **Check VM network references:**
```bash
# Check what networks the VMs are trying to use
kubectl get vm vm1 -o yaml | grep -A 5 "k8s.v1.cni.cncf.io/networks"
```

2. **List available NetworkAttachmentDefinitions:**
```bash
kubectl get network-attachment-definitions
```

3. **Fix the mismatch by either:**
   - Creating missing network definitions (e.g., if VMs reference `bridge-network1` but only `bridge-network10` exists)
   - Updating VM configurations to use existing networks

4. **Create missing bridge networks:**
```bash
# Example: Create bridge-network1 if VMs reference it
cat <<EOF | kubectl apply -f -
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: bridge-network1
spec:
  config: '{
    "cniVersion": "0.3.0",
    "name": "bridge-network1",
    "type": "bridge",
    "bridge": "br1",
    "ipam": {
      "type": "whereabouts",
      "range": "192.168.1.0/24",
      "network_name": "bridge-network1",
      "enable_overlapping_range": false
    }
  }'
EOF
```

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
# Delete individual VMs
kubectl delete vm vm1 vm2 vm3 vm4 vm5 vm6

# Or delete all VMs from the vm/ directory
kubectl delete -f vm/
```

To remove network configurations:

```bash
# Remove specific network attachments
kubectl delete -f nets/bridge-network10.yaml
kubectl delete -f nets/bridge-network11.yaml

# Or remove all network attachments
kubectl delete -f nets/
```

To remove EVPN/BGP infrastructure:

```bash
# Remove EVPN components
kubectl delete -f evpn/
```

To remove KubeVirt, Multus, and Whereabouts:

```bash
# To uninstall KubeVirt
export VERSION=v1.5.1
kubectl delete -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
kubectl delete -f https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml

# To uninstall Multus and Whereabouts
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/daemonset-install.yaml
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_ippools.yaml
kubectl delete -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml
```

To destroy the Talos cluster:

```bash
talosctl cluster destroy
```
