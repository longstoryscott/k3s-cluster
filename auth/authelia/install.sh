#!/usr/bin/env bash
set -e

SECRETS_DIR="${1}/.secrets"
mkdir -p "${SECRETS_DIR}"

function gen_secret() {
  local name=$1
  local value=$2
  local path="${SECRETS_DIR}/${name}"
  [[ -f "$path" ]] || echo "$value" >"$path"
}

kubectl create namespace auth || true

gen_secret jwt_secret "$(openssl rand -hex 32)"
gen_secret session_secret "$(openssl rand -hex 32)"
gen_secret storage_encryption_key "$(openssl rand -hex 32)"

kubectl create secret generic secrets \
  -n auth \
  --from-file=jwt_secret="${SECRETS_DIR}/jwt_secret" \
  --from-file=session_secret="${SECRETS_DIR}/session_secret" \
  --from-file=storage_encryption_key="${SECRETS_DIR}/storage_encryption_key" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n auth create configmap authelia-config \
  --from-file=configuration.yaml="${1}/configuration.yaml" \
  --from-file=users_database.yml="${1}/users_database.yml" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f "${1}/deployment.yaml"
kubectl rollout status deployment/authelia -n auth
kubectl apply -f "${1}/service.yaml"
