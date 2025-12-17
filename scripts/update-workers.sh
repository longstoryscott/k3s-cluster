#!/bin/bash

MASTER_IP=$(./set-static-ip.sh)
NODES=(lsnode-1.local lsnode-2.local lsnode-3)
CONF=$(sudo cat /etc/rancher/k3s/k3s.yaml | sed "s|127.0.0.1|${MASTER_IP}|g")

for node in "${NODES[@]}"; do
    ssh "${node}" "touch ~/.kube/config && \
        echo \"${CONF}\" > ~/.kube/config && \
        sudo chmod 600 ~/.kube/config && \
        sudo chown \"$USER\" ~/.kube/config && \
        touch /etc/rancher/k3s/k3s.yaml && \
        echo \"${CONF}\" > /etc/rancher/k3s/k3s.yaml && \
        sudo chmod 600 /etc/rancher/k3s/k3s.yaml && \
        sudo chown \"${USER}\" /etc/rancher/k3s/k3s.yaml"
done

# ../ingress/template.sh ../ingress/values.yaml /var/lib/rancher/k3s/server/manifests/ingress.yaml "$MASTER_IP"
