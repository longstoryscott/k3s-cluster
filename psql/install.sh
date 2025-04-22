#!/usr/bin/env bash
set -e

source "${1}/../helpers.sh"

kubectl create namespace psql || true
gen_secret psqlpw

kubectl create secret generic secrets \
    -n psql \
    "$(gen_secret psqlpw)" \
    --dry-run=client -o yaml | kubectl apply -f - --wait=true

# kubectl apply -n psql -f "${1}/pvc.yaml" --wait=true
kubectl apply -n psql -f "${1}/dbs.yaml" --wait=true
kubectl apply -n psql -f "${1}/statefulset.yaml" --wait=true
kubectl apply -n psql -f "${1}/service.yaml" --wait=true
kubectl rollout status statefulset/postgres -n psql
