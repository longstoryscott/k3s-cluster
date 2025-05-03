#!/bin/bash

TOKEN=sKrGps5Hfv5wB87obv5MHkhhM2fbFaPMSRrJc

MASTER_IP=$1
MASTER_NODE_NAME='lsnode-0'
NODES=(lsnode-1 lsnode-2 lsnode-3)

echo "setting ${MASTER_NODE_NAME} (${MASTER_IP}) as master node"

curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server --flannel-backend=vxlan --disable=traefik" K3S_NODE_NAME="${MASTER_NODE_NAME}" sh -s - --flannel-backend=vxlan --token ${TOKEN} --write-kubeconfig-mode 600
echo "installed k3s on ${MASTER_NODE_NAME} (${MASTER_IP})"
echo "setting up kubeconfig for ${MASTER_NODE_NAME} (${MASTER_IP})"
sudo chown "$USER" /etc/rancher/k3s/k3s.yaml
mkdir -p ~/.kube && touch ~/.kube/config
cat /etc/rancher/k3s/k3s.yaml >|~/.kube/config
sudo chmod 600 ~/.kube/config
CONF=$(sudo cat /etc/rancher/k3s/k3s.yaml | sed "s|127.0.0.1|${MASTER_IP}|g")

for node in "${NODES[@]}"; do
    ssh "${node}" "mkdir -p ~/.kube && touch ~/.kube/config && curl -sfL https://get.k3s.io | K3S_URL=https://${MASTER_IP}:6443 K3S_NODE_NAME=\"${node}\" K3S_TOKEN=${TOKEN} K3S_KUBECONFIG_MODE='600' sh -s - && echo \"${CONF}\" > ~/.kube/config && sudo chmod 600 ~/.kube/config"
done

# ../ingress/template.sh ../ingress/values.yaml /var/lib/rancher/k3s/server/manifests/ingress.yaml "$MASTER_IP"
