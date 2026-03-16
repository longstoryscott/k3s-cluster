#!/bin/bash

# This script forcefully unloads NVIDIA driver modules to allow the GPU Operator to install fresh drivers

set -e

echo "=== Unload Old NVIDIA Driver Modules ==="
echo ""
echo "This will:"
echo "1. Check for pods using GPU on lsnode-3"
echo "2. Forcefully unload old NVIDIA kernel modules"
echo "3. Allow GPU Operator to install fresh driver"
echo ""
echo "WARNING: This requires sudo access on lsnode-3"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 0
fi

echo ""
echo "Step 1: Checking for GPU workloads on lsnode-3..."

# Check for pods with nvidia runtime or GPU resources or NVIDIA_VISIBLE_DEVICES
GPU_PODS=$(kubectl get pods --all-namespaces -o json | jq -r '.items[] |
  select(.spec.nodeName == "lsnode-3") |
  select(
    (.spec.runtimeClassName == "nvidia") or
    (.spec.containers[]?.resources.limits."nvidia.com/gpu" != null) or
    (.spec.containers[]?.env[]? | select(.name == "NVIDIA_VISIBLE_DEVICES" and .value != "void"))
  ) |
  select(.metadata.namespace != "gpu-operator") |
"\(.metadata.namespace)/\(.metadata.name)"')

if [ -n "$GPU_PODS" ]; then
    echo "Found pods with GPU access on lsnode-3:"
    echo "$GPU_PODS"
    echo ""
    read -p "Delete these pods to proceed? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "  Deleting GPU pods..."
        for pod in $GPU_PODS; do
            echo "    Deleting $pod"
            kubectl delete pod -n ${pod%/*} ${pod##*/} --force --grace-period=0
        done
        echo "  Waiting 10 seconds for pods to terminate..."
        sleep 10
    else
        echo "Aborted. Please manually handle these pods."
        exit 1
    fi
else
    echo "  No GPU workloads found - safe to proceed"
fi

echo ""
echo "Step 2: Unloading NVIDIA kernel modules on lsnode-3..."
ssh lsm@lsnode-3.local 'sudo rmmod nvidia_uvm nvidia_drm nvidia_modeset nvidia 2>/dev/null || sudo modprobe -r nvidia_uvm nvidia_drm nvidia_modeset nvidia' || {
    echo ""
    echo "Failed to unload modules. The modules might be in use."
    echo "Checking what's holding them..."
    ssh lsm@lsnode-3.local 'lsmod | grep nvidia'
    echo ""
    echo "You may need to:"
    echo "  1. Reboot lsnode-3: ssh lsm@lsnode-3.local 'sudo reboot'"
    echo "  2. Or investigate what's using the modules"
    exit 1
}

echo "  Modules unloaded successfully"

echo ""
echo "Step 3: Verifying modules are unloaded..."
if ssh lsm@lsnode-3.local 'lsmod | grep -q nvidia'; then
    echo "ERROR: NVIDIA modules still loaded!"
    ssh lsm@lsnode-3 'lsmod | grep nvidia'
    exit 1
fi
echo "  Confirmed: No NVIDIA modules loaded"

echo ""
echo "Step 4: Restarting GPU Operator driver pod..."
kubectl delete pod -n gpu-operator -l app=nvidia-driver-daemonset --force --grace-period=0

echo ""
echo "=== Success ==="
echo ""
echo "The old driver modules have been unloaded."
echo "The GPU Operator should now be able to install its driver."
echo ""
echo "Monitor with:"
echo "  watch kubectl get pods -n gpu-operator"
