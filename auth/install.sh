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

# Create a combined ConfigMap for Dex templates with all necessary files and subdirectories
echo "Creating Dex templates ConfigMap with all files and subdirectories..."
WEB_DIR="$(dirname "$0")/web"

# First, create a temporary yaml file for the ConfigMap
TEMP_CM_FILE=$(mktemp)
echo "apiVersion: v1" >$TEMP_CM_FILE
echo "kind: ConfigMap" >>$TEMP_CM_FILE
echo "metadata:" >>$TEMP_CM_FILE
echo "  name: dex-templates" >>$TEMP_CM_FILE
echo "  namespace: auth" >>$TEMP_CM_FILE
echo "data:" >>$TEMP_CM_FILE

# Add robots.txt at the web root
if [ -f "$WEB_DIR/robots.txt" ]; then
  echo "  robots.txt: |" >>$TEMP_CM_FILE
  sed 's/^/    /' "$WEB_DIR/robots.txt" >>$TEMP_CM_FILE
fi

# Add web.go if it exists
if [ -f "$WEB_DIR/web.go" ]; then
  echo "  web.go: |" >>$TEMP_CM_FILE
  sed 's/^/    /' "$WEB_DIR/web.go" >>$TEMP_CM_FILE
fi

# Add files from the templates directory - use dots instead of slashes for keys
if [ -d "$WEB_DIR/templates" ]; then
  for file in "$WEB_DIR/templates"/*; do
    if [ -f "$file" ]; then
      filename=$(basename "$file")
      echo "  templates_$filename: |" >>$TEMP_CM_FILE
      sed 's/^/    /' "$file" >>$TEMP_CM_FILE
    fi
  done
fi

# Add files from the static directory - use dots instead of slashes for keys
if [ -d "$WEB_DIR/static" ]; then
  for file in "$WEB_DIR/static"/*; do
    if [ -f "$file" ]; then
      filename=$(basename "$file")
      echo "  static_$filename: |" >>$TEMP_CM_FILE
      sed 's/^/    /' "$file" >>$TEMP_CM_FILE
    fi
  done
fi

# Add files from the themes directory - use dots instead of slashes for keys
if [ -d "$WEB_DIR/themes" ]; then
  for file in "$WEB_DIR/themes"/*; do
    if [ -f "$file" ]; then
      filename=$(basename "$file")
      echo "  themes_$filename: |" >>$TEMP_CM_FILE
      sed 's/^/    /' "$file" >>$TEMP_CM_FILE
    fi
  done
fi

# Apply the ConfigMap
kubectl apply -f $TEMP_CM_FILE

# Clean up
rm $TEMP_CM_FILE

kubectl apply -f "$(dirname "$0")/deployment.yaml" --wait=true
kubectl rollout restart deployment/dex -n auth
kubectl rollout status deployment/dex -n auth
kubectl rollout status deployment/openldap -n auth
kubectl apply -f "$(dirname "$0")/service.yaml" --wait=true
kubectl apply -f "$(dirname "$0")/referencegrant.yaml" --wait=true

PW=$(kubectl get secret openldap-admin-secret -n auth -o jsonpath="{.data.LDAP_ADMIN_PASSWORD}" | base64 --decode)
kubectl exec -it deployment/openldap -n auth -- ldapadd -x -D "cn=admin,dc=longstorymedia,dc=com" -w "$PW" -f /ldif-seed/users.ldif || true
