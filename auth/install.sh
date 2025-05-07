#!/usr/bin/env bash
set -e

source "${1}/../helpers.sh"
kubectl create namespace auth || true

kubectl create secret generic secrets \
  -n auth \
  "$(gen_secret global_admin)" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap config \
  -n auth \
  --from-file="${1}/config.yaml" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

# kubectl apply -f "${1}/configmap.yaml" --wait=true
kubectl apply -f "${1}/deployment.yaml" --wait=true
kubectl rollout status deployment/dex -n auth
kubectl apply -f "${1}/service.yaml" --wait=true
kubectl apply -f "${1}/referencegrant.yaml" --wait=true
