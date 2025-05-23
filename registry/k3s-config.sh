#!/bin/bash

SERVER=$1
if [ -z "$SERVER" ]; then
  echo "Usage: $0 <server>"
  exit 1
fi

rsync -avz "$(dirname "$0")/k3s-config.yaml" "lsm@${SERVER}:/tmp/registries.yaml"
ssh "lsm@${SERVER}" 'sudo mkdir -p /etc/rancher/k3s && sudo touch /etc/rancher/k3s/registries.yaml && sudo mv /tmp/registries.yaml /etc/rancher/k3s/registries.yaml && sudo chown root:root /etc/rancher/k3s/registries.yaml && sudo systemctl restart k3s-agent || sudo systemctl restart k3s || true'
