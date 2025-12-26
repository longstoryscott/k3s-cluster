#!/bin/bash

# This script recovers from a failed driver upgrade by switching to pre-installed drivers

set -e

echo "=== NVIDIA GPU Operator Recovery ==="
echo ""
echo "This script will:"
echo "1. Clean up the failed driver upgrade state"
echo "2. Reconfigure the operator to use pre-installed host drivers"
echo "3. Restart the operator components"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
fi

echo ""
echo "Step 1: Removing failed driver pods..."
kubectl delete pod -n gpu-operator -l app=nvidia-driver-daemonset --force --grace-period=0 || true

echo ""
echo "Step 2: Clearing upgrade state labels from all GPU nodes..."
for node in $(kubectl get nodes -l nvidia.com/gpu.present -o name); do
    echo "  Cleaning $node..."
    kubectl label $node nvidia.com/gpu-driver-upgrade-state- --overwrite || true
    kubectl label $node nvidia.com/gpu-driver-upgrade.skip=true --overwrite
done

echo ""
echo "Step 3: Reconfiguring GPU Operator to use host drivers..."
helm upgrade --install nvidia-gpu-operator nvidia/gpu-operator \
  --namespace gpu-operator \
  --version=v25.10.1 \
  --set driver.enabled=false \
  --set toolkit.env[0].name=CONTAINERD_SOCKET \
  --set toolkit.env[0].value=/run/k3s/containerd/containerd.sock \
  --set toolkit.env[1].name=CONTAINERD_CONFIG \
  --set toolkit.env[1].value=/var/lib/rancher/k3s/agent/etc/containerd/config.toml \
  --set hostPaths.containerdSocket.path=/run/k3s/containerd \
  --wait

echo ""
echo "Step 4: Waiting for operator pods to stabilize..."
sleep 10

echo ""
echo "Step 5: Verifying deployment..."
kubectl get pods -n gpu-operator

echo ""
echo "=== Recovery Complete ==="
echo ""
echo "The GPU Operator is now configured to use pre-installed host drivers."
echo "Driver management by the operator is disabled."
echo ""
echo "To verify GPU access, run:"
echo "  kubectl run gpu-test --rm -it --restart=Never \\"
echo "    --image=nvidia/cuda:12.0.0-base-ubuntu22.04 \\"
echo "    --limits=nvidia.com/gpu=1 \\"
echo "    -- nvidia-smi"
echo ""
echo "Check pod status with:"
echo "  kubectl get pods -n gpu-operator"
