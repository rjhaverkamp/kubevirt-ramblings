# Complete Guide: Talos Cluster with Docker, KubeVirt, Multus, and VM Setup

This comprehensive guide covers everything you need to set up a Talos OS cluster using Docker, install KubeVirt and Multus CNI, and deploy virtual machines with multiple network interfaces. Multus CNI will handle the bridge creation automatically through network attachment definitions.

We will also add 2 vxlans, that will allow our multi-tenant virtual machines to communicate with each other. Even when they have overlapping address ranges.

## Table of Contents
- [Prerequisites](#prerequisites)
- [Step 1: Install Talos CLI Tool](#step-1-install-talos-cli-tool)
- [Step 2: Create a Talos Cluster with Docker](#step-2-create-a-talos-cluster-with-docker)
- [Step 3: Install KubeVirt, Multus CNI, and Whereabouts IPAM](#step-3-install-kubevirt-multus-cni-and-whereabouts-ipam)
- [Step 4: Configure Network Attachments](#step-4-configure-network-attachments)
- [Step 5: Create and Deploy VMs](#step-5-create-and-deploy-vms)
- [Step 6: VXLAN Setup](#step-6-vxlan-setup)
- [Step 7: FRR-k8s Setup](#step-7-frr-k8s-setup)
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
kubectl apply -f bridge-network.yaml
kubectl apply -f bridge-network2.yaml
```

Verify the network attachment definition:

```bash
kubectl get network-attachment-definitions
```

When these NetworkAttachmentDefinitions are applied, Multus CNI will automatically create and configure the bridge interfaces (`br1` and `br2`) on each node in the cluster. Unlike the manual approach, this method leverages Kubernetes' native networking capabilities, letting Multus handle the bridge creation and management automatically when VMs are scheduled.


## Step 5: Create and Deploy VMs

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

## Step 6: VXLAN Setup

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

## Step 7: FRR-k8s Setup

FRR-k8s is a Kubernetes-native FRR (Free Range Routing) daemon that can be used either as a standalone component or integrated with MetalLB. It provides advanced routing capabilities including BGP and EVPN in a Kubernetes environment.

### Install FRR-k8s

First, create the FRR-k8s namespace and install the CRDs:

```bash
# Create namespace
kubectl create namespace frr-k8s

# Clone the repository
git clone https://github.com/metallb/frr-k8s.git
cd frr-k8s

# Install CRDs
kubectl apply -f config/crd/bases/

# Install the controller
kubectl apply -f config/rbac
kubectl apply -f config/manager
```

Verify that the FRR-k8s controller is running:

```bash
kubectl get pods -n frr-k8s
```

### Configure FRR-k8s for Spine Setup

You can configure FRR-k8s using either the Kubernetes CRD-based approach or by using native FRR configuration syntax. We'll cover both methods:

#### Option 1: Using Kubernetes CRDs

Create a file named `frr-k8s-spine1.yaml` with the following configuration:

```bash
cat <<EOF > frr-k8s-spine1.yaml
apiVersion: frrk8s.metallb.io/v1beta1
kind: FRRConfiguration
metadata:
  name: spine1
  namespace: frr-k8s
spec:
  bgp:
    routers:
      - asn: 65000
        routerId: 1.1.1.11
        prefixes:
          - 1.1.1.11/32
        neighbors:
          - peerGroup: LEAF
            asn: external
            interfaces:
              - name: eth1
              - name: eth2
              - name: eth3
        peerGroups:
          - name: LEAF
            asn: external
            ebgpMultihop: true
            logNeighborChanges: true
        l2vpn:
          evpn: true
          clusterId: 1.1.1.11
          defaultGatewayPolicy: true
          neighbors:
            - peerGroup: FABRIC
              asn: 100
          peerGroups:
            - name: FABRIC
              asn: 100
              localAsn: 100
              updateSource: lo
              logNeighborChanges: true
              routeReflectorClient: true
        listenRanges:
          - peerGroup: FABRIC
            cidr: 1.1.1.0/24
        redistributeConnected: true
EOF
```

Deploy the FRR configuration to the control plane node:

```bash
kubectl apply -f frr-k8s-spine1.yaml
```

#### Option 2: Using Native FRR Configuration

For those familiar with FRR configuration syntax, you can use the native format. Create a file named `frr-native-config.yaml`:

```bash
cat <<EOF > frr-native-config.yaml
hostname spine1
!no ipv6 forwarding
!
interface lo
 ip address 1.1.1.11/32
exit
!
router bgp 65000
 bgp router-id 1.1.1.11
 bgp log-neighbor-changes
 bgp default l2vpn-evpn
 no bgp ebgp-requires-policy
 neighbor LEAF peer-group
 neighbor LEAF remote-as external
 neighbor eth1 interface peer-group LEAF
 neighbor eth2 interface peer-group LEAF
 neighbor eth3 interface peer-group LEAF
 !
 bgp cluster-id 1.1.1.11
 neighbor FABRIC peer-group
 neighbor FABRIC remote-as 100
 neighbor FABRIC local-as 100
 neighbor FABRIC update-source lo
 bgp listen range 1.1.1.0/24 peer-group FABRIC
 !
 address-family ipv4 unicast
  redistribute connected
 exit-address-family
 !
 address-family l2vpn evpn
  neighbor FABRIC activate
  neighbor FABRIC route-reflector-client
 exit-address-family
exit
!
EOF
```

Create a ConfigMap with this native configuration and apply it:

```bash
kubectl create configmap frr-config -n frr-k8s --from-file=frr.conf=frr-native-config.yaml
```

Update your FRR-k8s deployment to use this ConfigMap:

```bash
kubectl patch daemonset frr-node -n frr-k8s --type=json -p='[
  {
    "op": "add", 
    "path": "/spec/template/spec/volumes/1/configMap", 
    "value": {"name": "frr-config"}
  }
]'
```

Create a nodeSelector to ensure FRR runs only on the control plane node:

```bash
cat <<EOF > frr-k8s-deployment.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: frr-node
  namespace: frr-k8s
spec:
  selector:
    matchLabels:
      app: frr-node
  template:
    metadata:
      labels:
        app: frr-node
    spec:
      nodeSelector:
        node-role.kubernetes.io/control-plane: ""
      tolerations:
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule
      containers:
      - name: frr
        image: quay.io/metallb/frr:v7.5.1
        securityContext:
          privileged: true
        volumeMounts:
        - name: frr-socket
          mountPath: /var/run/frr
        - name: frr-conf
          mountPath: /etc/frr
      volumes:
      - name: frr-socket
        emptyDir: {}
      - name: frr-conf
        emptyDir: {}
EOF
```

Apply the deployment:

```bash
kubectl apply -f frr-k8s-deployment.yaml
```

### Verify FRR-k8s Setup

Check that the FRR container is running on the control plane node:

```bash
kubectl get pods -n frr-k8s -o wide
```

Verify the FRR configuration:

```bash
# Get the pod name
FRR_POD=$(kubectl get pods -n frr-k8s -l app=frr-node -o jsonpath='{.items[0].metadata.name}')

# View the running configuration
kubectl exec -it $FRR_POD -n frr-k8s -- vtysh -c "show running-config"

# Check BGP status
kubectl exec -it $FRR_POD -n frr-k8s -- vtysh -c "show bgp summary"

# Check L2VPN EVPN status
kubectl exec -it $FRR_POD -n frr-k8s -- vtysh -c "show bgp l2vpn evpn summary"

# Interactive FRR shell for advanced troubleshooting
kubectl exec -it $FRR_POD -n frr-k8s -- vtysh
```

### How It Integrates with Our Setup

FRR-k8s provides BGP routing capabilities which enhances our network setup by:

1. **Route Advertisement**: Automatically advertises pod and service CIDRs to external routers
2. **EVPN Support**: Enables VXLAN control plane for more robust multi-tenant networking
3. **High Availability**: Ensures proper failover and load balancing between nodes

This BGP setup configures the control plane node as a route reflector (RR) with the router ID 1.1.1.11, which helps in scaling BGP by reducing the number of peering sessions required.

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

1. Verify Multus, Whereabouts, and FRR-k8s are running:
```bash
kubectl get pods -n kube-system | grep -E 'multus|whereabouts'
kubectl get pods -n frr-k8s
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

6. Verify FRR-k8s BGP status:
```bash
FRR_POD=$(kubectl get pods -n frr-k8s -l app=frr-node -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $FRR_POD -n frr-k8s -- vtysh -c "show bgp summary"
kubectl exec -it $FRR_POD -n frr-k8s -- vtysh -c "show bgp l2vpn evpn summary"
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

# To uninstall FRR-k8s
kubectl delete -f frr-k8s-deployment.yaml
kubectl delete -f frr-k8s-spine1.yaml
kubectl delete configmap frr-config -n frr-k8s
kubectl delete -f config/manager -n frr-k8s
kubectl delete -f config/rbac -n frr-k8s
kubectl delete -f config/crd/bases/
kubectl delete namespace frr-k8s
```

To destroy the Talos cluster:

```bash
talosctl cluster destroy
```
