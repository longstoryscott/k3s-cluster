#!/bin/bash
set -e

echo "=== NVIDIA Driver Downgrade: 580.105.08 → 580.95.05 ==="
echo ""
echo "This will downgrade the driver using the GPU Operator"
echo "The node will be drained and rebooted automatically"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

echo ""
echo "Step 1: Disabling auto-upgrade..."
kubectl patch clusterpolicies.nvidia.com cluster-policy --type='json' \
  -p='[{"op": "replace", "path": "/spec/driver/upgradePolicy/autoUpgrade", "value": false}]'

echo ""
echo "Step 2: Setting driver version to 580.95.05..."
kubectl patch clusterpolicies.nvidia.com cluster-policy --type='json' \
  -p='[{"op": "replace", "path": "/spec/driver/version", "value": "580.95.05"}]'

echo ""
echo "Step 3: Marking node for downgrade..."
kubectl label node lsnode-3 nvidia.com/gpu-driver-upgrade-state=upgrade-required --overwrite

echo ""
echo "Step 4: Monitoring downgrade progress..."
echo "Expected states: upgrade-required → cordon-required → pod-deletion-required → pod-restart-required → validation-required → uncordon-required → upgrade-done"
echo ""
echo "Press Ctrl+C to stop monitoring (downgrade will continue in background)"
echo ""

watch -n 3 'echo "=== Node State ===" && \
kubectl get node lsnode-3 -o jsonpath="Upgrade State: {.metadata.labels.nvidia\.com/gpu-driver-upgrade-state}{\"\\n\"}" && \
echo "" && \
echo "=== Driver Pods ===" && \
kubectl get pods -n gpu-operator -l app=nvidia-driver-daemonset -o wide | grep lsnode-3 && \
echo "" && \
echo "=== Current Driver Version ===" && \
kubectl exec -n steam selkies-desktop-official-0 -- nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null || echo "Pod not running"'
