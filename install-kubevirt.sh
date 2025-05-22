#!/bin/bash
set -e

# Create the necessary namespace
kubectl create namespace kubevirt 2>/dev/null || echo "Namespace kubevirt already exists"

# Use a specific stable KubeVirt version that's known to work
export VERSION=v1.5.1

# Force direct download with wget instead of curl
echo "Downloading KubeVirt operator manifest..."
wget -q -O kubevirt-operator.yaml https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-operator.yaml
kubectl apply -f kubevirt-operator.yaml
rm kubevirt-operator.yaml

echo "Downloading KubeVirt custom resource manifest..."
wget -q -O kubevirt-cr.yaml https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/kubevirt-cr.yaml
kubectl apply -f kubevirt-cr.yaml
rm kubevirt-cr.yaml

echo "Waiting for KubeVirt to become ready..."
kubectl -n kubevirt wait kv kubevirt --for condition=Available --timeout=300s

echo "Installing virtctl tool..."
if [ ! -f /usr/local/bin/virtctl ]; then
  wget -q -O virtctl https://github.com/kubevirt/kubevirt/releases/download/${VERSION}/virtctl-${VERSION}-linux-amd64
  chmod +x virtctl
  sudo mv virtctl /usr/local/bin/ || mv virtctl $HOME/bin/ || echo "Failed to install virtctl to a system path. It's available in the current directory."
fi

echo "KubeVirt installation complete!"

# Install Multus CNI
echo "Installing Multus CNI..."
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset.yml

echo "Waiting for Multus CNI to be ready..."
kubectl rollout status -n kube-system daemonset/kube-multus-ds --timeout=180s

# Install Whereabouts IPAM
echo "Installing Whereabouts IPAM..."
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/daemonset-install.yaml

# Install Whereabouts CRDs
echo "Installing Whereabouts CRDs..."
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_ippools.yaml
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/whereabouts/master/doc/crds/whereabouts.cni.cncf.io_overlappingrangeipreservations.yaml

echo "Waiting for Whereabouts to be ready..."
kubectl rollout status -n kube-system daemonset/whereabouts --timeout=180s

echo "Multus CNI and Whereabouts IPAM installation complete!"