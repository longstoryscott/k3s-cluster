#!/bin/bash

# This script cleans up the failed driver state and reinstalls the GPU Operator
# with proper driver management for dedicated k8s nodes

set -e

echo "=== NVIDIA GPU Operator Cleanup and Reinstall ==="
echo ""
echo "This will:"
echo "1. Remove all failed/stuck GPU operator pods"
echo "2. Clear upgrade state labels"  
echo "3. Unload any stuck driver modules on the host"
echo "4. Reinstall with driver management enabled"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
fi

echo ""
echo "Step 1: Deleting all GPU operator pods..."
kubectl delete pods -n gpu-operator --all --force --grace-period=0 || true
sleep 5

echo ""
echo "Step 2: Clearing upgrade state from GPU nodes..."
for node in $(kubectl get nodes -l nvidia.com/gpu.present -o jsonpath='{.items[*].metadata.name}'); do
    echo "  Cleaning $node..."
    kubectl label node $node nvidia.com/gpu-driver-upgrade-state- --overwrite 2>/dev/null || true
    kubectl label node $node nvidia.com/gpu-driver-upgrade.skip- --overwrite 2>/dev/null || true
done

echo ""
echo "Step 3: Unloading any stuck NVIDIA driver modules on lsnode-3..."
ssh lsm@lsnode-3 'sudo rmmod nvidia_uvm nvidia_drm nvidia_modeset nvidia 2>/dev/null || echo "  No modules to unload or already unloaded"'

echo ""
echo "Step 4: Reinstalling GPU Operator with driver management..."
helm upgrade --install nvidia-gpu-operator nvidia/gpu-operator \
  --namespace gpu-operator \
  --version=v25.10.1 \
  --set driver.enabled=true \
  --set driver.upgradePolicy.autoUpgrade=true \
  --set driver.upgradePolicy.drain.enable=true \
  --set driver.upgradePolicy.drain.force=true \
  --set driver.upgradePolicy.drain.deleteEmptyDir=true \
  --set driver.upgradePolicy.gpuPodDeletion.force=true \
  --set driver.upgradePolicy.gpuPodDeletion.deleteEmptyDir=true \
  --set toolkit.env[0].name=CONTAINERD_SOCKET \
  --set toolkit.env[0].value=/run/k3s/containerd/containerd.sock \
  --set toolkit.env[1].name=CONTAINERD_CONFIG \
  --set toolkit.env[1].value=/var/lib/rancher/k3s/agent/etc/containerd/config.toml \
  --set hostPaths.containerdSocket.path=/run/k3s/containerd \
  --wait

echo ""
echo "Step 5: Waiting for pods to start..."
sleep 10

echo ""
echo "Step 6: Checking pod status..."
kubectl get pods -n gpu-operator

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Monitor the driver installation with:"
echo "  watch kubectl get pods -n gpu-operator"
echo ""
echo "Check driver upgrade status with:"
echo "  kubectl get node -l nvidia.com/gpu.present -o jsonpath='{range .items[*]}{.metadata.name}{\"\\t\"}{.metadata.labels.nvidia\\.com/gpu-driver-upgrade-state}{\"\\n\"}{end}'"
echo ""
echo "Once all pods are Running, test GPU access with:"
echo "  kubectl run gpu-test --rm -it --restart=Never --image=nvidia/cuda:12.0.0-base-ubuntu22.04 --limits=nvidia.com/gpu=1 -- nvidia-smi"
