#!/bin/bash

helm template ingress-nginx ingress-nginx \
    --repo https://kubernetes.github.io/ingress-nginx \
    --set controller.service.externalIPs={"${3}"} \
    --namespace ingress-nginx --create-namespace --atomic \
    --values "${1}" | sudo tee "${2}" || {
    echo "failed to generate manifest! something wrong with the helm chart?"
    exit 1
}
