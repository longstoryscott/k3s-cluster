#!/bin/bash

# Fix NVIDIA device plugin stuck in Init phase
# This happens when the container runtime hasn't restarted after NVIDIA toolkit installation

set -e

echo "=== NVIDIA Device Plugin Init Fix ==="
echo ""
echo "Problem: Device plugin and other GPU operator pods stuck in Init phase"
echo "Root cause: Container runtime needs to reload NVIDIA runtime configuration"
echo ""
echo "This script will:"
echo "1. Restart k3s-agent on lsnode-3 to reload containerd with NVIDIA runtime"
echo "2. Wait for node to become ready"
echo "3. Verify GPU resources are advertised"
echo "4. Monitor pod initialization"
echo ""
read -p "Continue? (y/n): " confirm

if [[ "$confirm" != "y" ]]; then
  echo "Aborted."
  exit 0
fi

echo ""
echo "Step 1: Restarting k3s-agent on lsnode-3..."
ssh lsnode-3 "sudo systemctl restart k3s-agent"

echo "Waiting for node to restart (30s)..."
sleep 30

echo ""
echo "Step 2: Waiting for node to become Ready..."
for i in {1..30}; do
  STATUS=$(kubectl get node lsnode-3 -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')
  if [[ "$STATUS" == "True" ]]; then
    echo "✓ Node lsnode-3 is Ready"
    break
  fi
  echo "Waiting for node... ($i/30)"
  sleep 5
done

if [[ "$STATUS" != "True" ]]; then
  echo "⚠ Warning: Node not ready after 2.5 minutes"
  echo "Check with: kubectl get nodes"
  exit 1
fi

echo ""
echo "Step 3: Checking GPU resources..."
sleep 10
GPU_COUNT=$(kubectl get node lsnode-3 -o jsonpath='{.status.capacity.nvidia\.com/gpu}')

if [[ -z "$GPU_COUNT" || "$GPU_COUNT" == "0" ]]; then
  echo "⚠ Warning: GPUs not yet advertised by node"
  echo "Current GPU count: ${GPU_COUNT:-none}"
  echo ""
  echo "This may take a few more minutes. Monitor with:"
  echo "  watch -n 5 'kubectl get node lsnode-3 -o jsonpath=\"{.status.capacity}\" | jq'"
else
  echo "✓ Node advertising $GPU_COUNT GPUs"
fi

echo ""
echo "Step 4: Monitoring GPU operator pods..."
echo "Waiting for device plugin to initialize (up to 2 minutes)..."

for i in {1..24}; do
  DEVICE_PLUGIN_READY=$(kubectl get pods -n gpu-operator -l app=nvidia-device-plugin-daemonset -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null || echo "false")
  
  if [[ "$DEVICE_PLUGIN_READY" == "true" ]]; then
    echo "✓ Device plugin is ready!"
    break
  fi
  
  INIT_STATUS=$(kubectl get pods -n gpu-operator -l app=nvidia-device-plugin-daemonset -o jsonpath='{.items[0].status.initContainerStatuses[0].state}' 2>/dev/null || echo "unknown")
  echo "[$i/24] Device plugin status: $INIT_STATUS"
  sleep 5
done

echo ""
echo "Current GPU operator pod status:"
kubectl get pods -n gpu-operator | grep -E "(NAME|device-plugin|gpu-feature|dcgm|validator)"

echo ""
echo "=== Summary ==="
kubectl get node lsnode-3 -o jsonpath='Node: {.metadata.name}{"\n"}GPUs: {.status.capacity.nvidia\.com/gpu}{"\n"}'

echo ""
echo "If pods are still in Init phase, check logs:"
echo "  kubectl logs -n gpu-operator <pod-name> -c toolkit-validation"
echo ""
echo "To monitor Ollama pod:"
echo "  kubectl get pods -n ai -w"
