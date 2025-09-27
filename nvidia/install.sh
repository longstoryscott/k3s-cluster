#!/bin/bash

# This script installs the NVIDIA driver and CUDA toolkit on Ubuntu 22.04.

kubectl label --overwrite ns gpu-operator pod-security.kubernetes.io/enforce=privileged
helm repo add nvidia https://nvidia.github.io/gpu-operator
helm repo update
# Ensure namespace exists and is labeled before Helm install
kubectl create namespace gpu-operator --dry-run=client -o yaml | kubectl apply -f -
kubectl label --overwrite ns gpu-operator pod-security.kubernetes.io/enforce=privileged
helm upgrade --install nvidia-gpu-operator nvidia/gpu-operator \
  --namespace gpu-operator \
  --version=v25.3.2 \
  --set toolkit.env[0].name=CONTAINERD_SOCKET \
  --set toolkit.env[0].value=/run/k3s/containerd/containerd.sock \
  --set toolkit.env[1].name=CONTAINERD_CONFIG \
  --set toolkit.env[1].value=/var/lib/rancher/k3s/agent/etc/containerd/config.toml \
  --set hostPaths.containerdSocket.path=/run/k3s/containerd
  # --wait \
# --values $(dirname $0)/values.yaml
