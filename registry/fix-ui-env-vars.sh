#!/usr/bin/env bash

echo "=== Fixing Registry UI Environment Variables ==="

# Get credentials
SECRETS_DIR="$(dirname "$0")/.secrets"
USER_SECRET_FILE="${SECRETS_DIR}/registryuser"
PW_SECRET_FILE="${SECRETS_DIR}/registrypw"

if [[ -f "$USER_SECRET_FILE" && -f "$PW_SECRET_FILE" ]]; then
    REGISTRY_USER=$(cat "$USER_SECRET_FILE")
    REGISTRY_PW=$(cat "$PW_SECRET_FILE")
else
    REGISTRY_USER="dc4c537fa4a4acf4b2906b94"
    REGISTRY_PW="613a048331b0928c6ae3b0aa"
fi

AUTH_STRING=$(echo -n "${REGISTRY_USER}:${REGISTRY_PW}" | base64)

echo "Updating deployment with correct environment variables..."

# Replace the entire deployment with correct env vars
kubectl patch deployment registry-ui -n registry --type='merge' -p="{
  \"spec\": {
    \"template\": {
      \"spec\": {
        \"containers\": [{
          \"name\": \"registry-ui\",
          \"image\": \"joxit/docker-registry-ui:static\",
          \"imagePullPolicy\": \"IfNotPresent\",
          \"env\": [
            {\"name\": \"REGISTRY_TITLE\", \"value\": \"Private Registry UI\"},
            {\"name\": \"DELETE_IMAGES\", \"value\": \"true\"},
            {\"name\": \"REGISTRY_URL\", \"value\": \"http://registry.registry.svc.cluster.local:5000\"},
            {\"name\": \"PULL_URL\", \"value\": \"registry.local:31500\"},
            {\"name\": \"SINGLE_REGISTRY\", \"value\": \"true\"},
            {\"name\": \"CATALOG_ELEMENTS_LIMIT\", \"value\": \"1000\"},
            {\"name\": \"SHOW_CONTENT_DIGEST\", \"value\": \"true\"},
            {\"name\": \"THEME\", \"value\": \"auto\"},
            {\"name\": \"NGINX_PROXY_PASS_URL\", \"value\": \"http://registry.registry.svc.cluster.local:5000\"},
            {\"name\": \"NGINX_PROXY_HEADER_Authorization\", \"value\": \"Basic ${AUTH_STRING}\"}
          ],
          \"ports\": [{\"name\": \"http\", \"containerPort\": 80, \"protocol\": \"TCP\"}],
          \"livenessProbe\": {
            \"httpGet\": {\"path\": \"/\", \"port\": \"http\"},
            \"initialDelaySeconds\": 10,
            \"timeoutSeconds\": 5,
            \"periodSeconds\": 10
          },
          \"readinessProbe\": {
            \"httpGet\": {\"path\": \"/\", \"port\": \"http\"},
            \"initialDelaySeconds\": 5,
            \"timeoutSeconds\": 3,
            \"periodSeconds\": 5
          },
          \"resources\": {
            \"limits\": {\"cpu\": \"100m\", \"memory\": \"128Mi\"},
            \"requests\": {\"cpu\": \"50m\", \"memory\": \"64Mi\"}
          }
        }]
      }
    }
  }
}"

echo "Waiting for deployment to roll out..."
kubectl rollout status deployment/registry-ui -n registry --timeout=120s

echo "Testing the configuration..."
sleep 10

NEW_UI_POD=$(kubectl get pods -n registry -l app=registry-ui -o jsonpath='{.items[0].metadata.name}')
if [ -n "${NEW_UI_POD}" ]; then
    echo "=== New nginx configuration ==="
    kubectl exec "${NEW_UI_POD}" -n registry -- cat /etc/nginx/conf.d/default.conf | grep -A 10 -B 2 "location /v2"
    
    echo "=== Environment variables ==="
    kubectl exec "${NEW_UI_POD}" -n registry -- env | grep -E "(REGISTRY|NGINX)" | sort
    
    echo "=== Testing connection ==="
    kubectl exec "${NEW_UI_POD}" -n registry -- curl -s -o /dev/null -w "%{http_code}" -u "${REGISTRY_USER}:${REGISTRY_PW}" http://registry.registry.svc.cluster.local:5000/v2/_catalog
fi

NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
echo
echo "Registry UI should now be accessible at: http://${NODE_IP}:31501"