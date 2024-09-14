#! /bin/bash

CWD=$1
PATCH=$2

if [ "${PATCH}" = "patch" ]; then
    source "${CWD}/accessmgmt.sh"

    registry-init-secrets

    echo 'enter username'

    read -r un

    echo 'enter password'

    read -s -r pw

    registry-add-user "$un" "$pw"

    generate-cert

    kubectl apply -f "${CWD}/manifest.yaml"

    k8s-secrets
else
    kubectl apply -f "${CWD}/manifest.yaml"
fi
