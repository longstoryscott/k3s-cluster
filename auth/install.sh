#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../helpers.sh"
kubectl create namespace auth || true

kubectl create secret generic secrets \
  -n auth \
  "$(gen_secret global_admin)" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create secret generic openldap-admin-secret \
  -n auth \
  --from-file=LDAP_ADMIN_PASSWORD="$(dirname "$0")/.secrets/lsm_secret" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap config \
  -n auth \
  --from-file="$(dirname "$0")/config.yaml" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap openldap-seed-ldif \
  -n auth \
  --from-file="$(dirname "$0")/users.ldif" \
  --from-file="$(dirname "$0")/groups.ldif" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl apply -f "$(dirname "$0")/deployment.yaml" --wait=true
kubectl rollout status deployment/dex -n auth
kubectl apply -f "$(dirname "$0")/service.yaml" --wait=true
kubectl apply -f "$(dirname "$0")/referencegrant.yaml" --wait=true

PW=$(kubectl get secret openldap-admin-secret -n auth -o jsonpath="{.data.LDAP_ADMIN_PASSWORD}" | base64 --decode)
kubectl exec -it deployment/openldap -n auth -- ldapadd -x -D "cn=admin,dc=longstorymedia,dc=com" -w "$PW" -f /ldif-seed/users.ldif || true
