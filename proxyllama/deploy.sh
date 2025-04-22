#!/bin/bash

source "${1}/../helpers.sh"

read -sp "Enter sudo password: " PW
echo

set -e

kubectl create namespace proxyllama || true

kubectl create secret generic secrets \
    -n proxyllama \
    --from-file=psqlpw="${1}/../psql/.secrets/psqlpw" \
    --dry-run=client -o yaml | kubectl apply -f - --wait=true

NODES=(lsnode-0.local lsnode-1.local lsnode-2.local)

docker build -t proxyllama:latest "${1}"
docker save proxyllama:latest -o "${1}/proxyllama.tar"

for node in "${NODES[@]}"; do
    ssh "lsm@${node}" "
        if [ ! -d ~/builds ]; then
            mkdir -p ~/builds
        fi

        if [ ! -d ~/builds/proxyllama ]; then
            mkdir -p ~/builds/proxyllama
        fi
    "
    scp "${1}/proxyllama.tar" "lsm@${node}:~/builds/proxyllama/proxyllama.tar"

    ssh "lsm@${node}" "echo '${PW}' | sudo -S ctr -n=proxyllama images import ~/builds/proxyllama/proxyllama.tar && 
        echo '${PW}' | sudo -S ctr images tag docker.io/library/proxyllama:latest proxyllama:latest"

done

kubectl apply -n proxyllama -f "${1}/k8s/deployment.yaml" --wait=true
kubectl apply -n proxyllama -f "${1}/k8s/service.yaml" --wait=true
