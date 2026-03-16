#!/bin/bash

# This script installs the NVIDIA GPU Operator for k3s
# The operator will manage drivers in containers (no host driver installation needed)

kubectl label --overwrite ns gpu-operator pod-security.kubernetes.io/enforce=privileged
helm repo add nvidia https://nvidia.github.io/gpu-operator
helm repo update
# Ensure namespace exists and is labeled before Helm install
kubectl create namespace gpu-operator --dry-run=client -o yaml | kubectl apply -f -
kubectl label --overwrite ns gpu-operator pod-security.kubernetes.io/enforce=privileged
helm upgrade --install nvidia-gpu-operator nvidia/gpu-operator \
--namespace gpu-operator \
--version=v25.10.1 \
--set driver.enabled=true \
--set driver.upgradePolicy.autoUpgrade=true \
--set driver.upgradePolicy.drain.enable=true \
--set driver.upgradePolicy.drain.force=true \
--set driver.upgradePolicy.drain.deleteEmptyDir=true \
--set toolkit.env[0].name=CONTAINERD_SOCKET \
--set toolkit.env[0].value=/run/k3s/containerd/containerd.sock \
--set toolkit.env[1].name=CONTAINERD_CONFIG \
--set toolkit.env[1].value=/var/lib/rancher/k3s/agent/etc/containerd/config.toml \
--set toolkit.env[2].name=ENABLE_AUTO_DRAIN \
--set toolkit.env[2].value=true \
--set hostPaths.containerdSocket.path=/run/k3s/containerd
# --wait \
# --values $(dirname $0)/values.yaml
