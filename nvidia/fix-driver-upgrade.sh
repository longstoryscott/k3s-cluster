#!/bin/bash

# This script provides options to fix the NVIDIA driver upgrade failure

echo "=== NVIDIA Driver Upgrade Fix ==="
echo ""
echo "Current situation:"
echo "- Node lsnode-3 is in 'upgrade-failed' state"
echo "- Current driver: 570.148.08"
echo "- Target driver: 580.105.08"
echo ""
echo "Choose an option:"
echo "1) Retry upgrade to 580.105.08 (recommended)"
echo "2) Disable auto-upgrade and keep 570.148.08"
echo "3) Skip lsnode-3 from future upgrades"
echo "4) Check current upgrade status"
echo "5) View operator logs"
read -p "Enter option (1-5): " option

case $option in
  1)
    echo "Resetting lsnode-3 to 'upgrade-required' state..."
    kubectl label node lsnode-3 nvidia.com/gpu-driver-upgrade-state=upgrade-required --overwrite
    echo ""
    echo "Monitoring upgrade progress (Ctrl+C to exit)..."
    echo "Expected states: upgrade-required -> cordon-required -> pod-deletion-required -> pod-restart-required -> validation-required -> uncordon-required -> upgrade-done"
    echo ""
    watch -n 2 'kubectl get node lsnode-3 -o jsonpath="{.metadata.labels.nvidia\.com/gpu-driver-upgrade-state}" && echo "" && kubectl get pods -n gpu-operator | grep driver'
    ;;
  2)
    echo "Disabling auto-upgrade..."
    kubectl patch clusterpolicies.nvidia.com/cluster-policy --type='json' \
      -p='[{"op": "replace", "path": "/spec/driver/upgradePolicy/autoUpgrade", "value":false}]'
    echo "Updating driver version to current: 570.148.08"
    kubectl patch clusterpolicies.nvidia.com/cluster-policy --type='json' \
      -p='[{"op": "replace", "path": "/spec/driver/version", "value":"570.148.08"}]'
    echo "Clearing upgrade-failed state..."
    kubectl label node lsnode-3 nvidia.com/gpu-driver-upgrade-state- --overwrite
    echo ""
    echo "Done! Auto-upgrade disabled, driver version set to 570.148.08"
    ;;
  3)
    echo "Marking lsnode-3 to skip future upgrades..."
    kubectl label node lsnode-3 nvidia.com/gpu-driver-upgrade.skip=true
    echo "Clearing upgrade-failed state..."
    kubectl label node lsnode-3 nvidia.com/gpu-driver-upgrade-state- --overwrite
    echo ""
    echo "Done! lsnode-3 will skip future driver upgrades"
    ;;
  4)
    echo "=== Current Upgrade Status ==="
    echo ""
    echo "Node upgrade states:"
    kubectl get node -l nvidia.com/gpu.present \
      -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.labels.nvidia\.com/gpu-driver-upgrade-state}{"\n"}{end}'
    echo ""
    echo "ClusterPolicy status:"
    kubectl get clusterpolicies.nvidia.com/cluster-policy -o jsonpath='{.status.state}{"\n"}'
    echo ""
    echo "Driver pods:"
    kubectl get pods -n gpu-operator -l app=nvidia-driver-daemonset
    echo ""
    echo "Recent GPU driver upgrade events:"
    kubectl get events --all-namespaces --sort-by='.lastTimestamp' | grep GPUDriverUpgrade | tail -10
    ;;
  5)
    echo "=== GPU Operator Logs (last 50 lines) ==="
    kubectl logs -n gpu-operator deployment/gpu-operator --tail=50 | grep -i "upgrade\|driver\|error"
    ;;
  *)
    echo "Invalid option"
    exit 1
    ;;
esac
