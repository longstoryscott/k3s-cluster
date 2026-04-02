#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../helpers.sh"

# Use registry.local instead of NODE_IP:PORT
REGISTRY_HOME="$(dirname "$0")/../registry"
REGISTRY_URL="192.168.0.71:31500"

# Registry credentials
USER_SECRET_FILE="${REGISTRY_HOME}/.secrets/registryuser"
PW_SECRET_FILE="${REGISTRY_HOME}/.secrets/registrypw"

if [[ -f "$USER_SECRET_FILE" && -f "$PW_SECRET_FILE" ]]; then
    REGISTRY_USER=$(cat "$USER_SECRET_FILE")
    REGISTRY_PW=$(cat "$PW_SECRET_FILE")
else
    echo "Registry credentials not found. Please run registry-mgmt.sh install first."
    exit 1
fi

kubectl create namespace auth || true

# Create registry credentials secret
kubectl create secret docker-registry registry-credentials \
-n auth \
--docker-server="${REGISTRY_URL}" \
--docker-username="${REGISTRY_USER}" \
--docker-password="${REGISTRY_PW}" \
--dry-run=client -o yaml | kubectl apply -f - --wait=true

echo "Building and pushing image to private registry..."
bash "$(dirname "$0")/build-push.sh" "${VERSION}"

# Add a log message to see what's happening with the secret
echo "Creating client-secret with key: client-secret"
kubectl create secret generic client-secret \
-n auth \
"$(gen_secret client-secret)" \
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
--from-file="$(dirname "$0")/ldap-users.ldif" \
--from-file="$(dirname "$0")/ldap-groups.ldif" \
--dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap dex-web \
-n auth \
--from-file="$(dirname "$0")/web" \
--dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap dex-templates \
-n auth \
--from-file="$(dirname "$0")/web/templates" \
--dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap dex-static \
-n auth \
--from-file="$(dirname "$0")/web/static" \
--dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl create configmap dex-themes \
-n auth \
--from-file="$(dirname "$0")/web/themes" \
--dry-run=client -o yaml | kubectl apply -f - --wait=true

kubectl apply -f "$(dirname "$0")/pvc.yaml" --wait=true
kubectl apply -f "$(dirname "$0")/deployment.yaml" --wait=true
kubectl rollout restart deployment/dex -n auth
kubectl rollout restart deployment/openldap -n auth
kubectl rollout restart deployment/usrmgr -n auth
kubectl rollout status deployment/dex -n auth
kubectl rollout status deployment/openldap -n auth
kubectl rollout status deployment/usrmgr -n auth
kubectl apply -f "$(dirname "$0")/service.yaml" --wait=true
kubectl apply -f "$(dirname "$0")/referencegrant.yaml" --wait=true

PW=$(kubectl get secret openldap-admin-secret -n auth -o jsonpath="{.data.LDAP_ADMIN_PASSWORD}" | base64 --decode)
kubectl exec -it deployment/openldap -n auth -- ldapadd -x -D "cn=admin,dc=longstorymedia,dc=com" -w "$PW" -f /ldif-seed/ldap-users.ldif || true
kubectl exec -it deployment/openldap -n auth -- ldapadd -x -D "cn=admin,dc=longstorymedia,dc=com" -w "$PW" -f /ldif-seed/ldap-groups.ldif || true
