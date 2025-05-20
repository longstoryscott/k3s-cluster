#!/bin/bash

# This script installs the NVIDIA driver and CUDA toolkit on Ubuntu 22.04.

helm repo add nvidia https://nvidia.github.io/gpu-operator
helm repo update
helm upgrade --install nvidia-gpu-operator nvidia/gpu-operator \
  --namespace gpu-operator \
  --create-namespace \
  --wait \
  --set toolkit.env[0].name=CONTAINERD_SOCKET \
  --set toolkit.env[0].value=/run/k3s/containerd/containerd.sock \
  --set toolkit.env[1].name=CONTAINERD_CONFIG \
  --set toolkit.env[1].value=/var/lib/rancher/k3s/agent/etc/containerd/config.toml \
  --set hostPaths.containerdSocket.path=/run/k3s/containerd
# --values $(dirname $0)/values.yaml

kubectl label --overwrite ns gpu-operator pod-security.kubernetes.io/enforce=privileged
