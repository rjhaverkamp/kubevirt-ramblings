#!/bin/bash
set -e

# Create the necessary namespace
kubectl create namespace kubevirt 2>/dev/null || echo "Namespace kubevirt already exists"
3veth0ee934c3veth0ee934c3veth0ee934bjjxhnhtnydngpckjbdxcjncu
c
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