#!/usr/bin/env bash

echo "=== Debugging UI Connection ==="
echo

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

echo "Using credentials: ${REGISTRY_USER}:${REGISTRY_PW}"
echo

# Check current UI deployment env vars
echo "1. Current UI deployment environment variables:"
kubectl get deployment registry-ui -n registry -o jsonpath='{.spec.template.spec.containers[0].env[*]}' | jq '.'
echo

# Test direct connection from UI pod to registry
UI_POD=$(kubectl get pods -n registry -l app=registry-ui -o jsonpath='{.items[0].metadata.name}')
REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}')

echo "2. Testing connection from UI pod to registry service:"
if [ -n "${UI_POD}" ]; then
    echo "Testing without auth:"
    kubectl exec "${UI_POD}" -n registry -- curl -s -o /dev/null -w "%{http_code}" http://registry.registry.svc.cluster.local:5000/v2/ || echo "Connection failed"
    
    echo "Testing with auth:"
    kubectl exec "${UI_POD}" -n registry -- curl -s -o /dev/null -w "%{http_code}" -u "${REGISTRY_USER}:${REGISTRY_PW}" http://registry.registry.svc.cluster.local:5000/v2/ || echo "Connection failed"
    
    echo "Testing catalog endpoint:"
    kubectl exec "${UI_POD}" -n registry -- curl -s -u "${REGISTRY_USER}:${REGISTRY_PW}" http://registry.registry.svc.cluster.local:5000/v2/_catalog || echo "Catalog failed"
else
    echo "UI pod not found"
fi
echo

# Check if registry is actually responding
echo "3. Testing registry directly:"
if [ -n "${REGISTRY_POD}" ]; then
    echo "Registry internal health:"
    kubectl exec "${REGISTRY_POD}" -n registry -- curl -s -o /dev/null -w "%{http_code}" http://localhost:5000/v2/ || echo "Registry internal check failed"
else
    echo "Registry pod not found"
fi
echo

# Check UI logs for any errors
echo "4. Recent UI logs (looking for errors):"
if [ -n "${UI_POD}" ]; then
    kubectl logs "${UI_POD}" -n registry --tail=20 | grep -E "(error|Error|ERROR|fail|Fail|FAIL)" || echo "No obvious errors in logs"
else
    echo "No UI pod found"
fi
echo

# Check browser console by examining nginx access logs more carefully
echo "5. Recent UI access logs:"
if [ -n "${UI_POD}" ]; then
    kubectl logs "${UI_POD}" -n registry --tail=10
fi
echo

# Test the UI's nginx config
echo "6. Check UI nginx configuration:"
if [ -n "${UI_POD}" ]; then
    echo "Nginx config snippet:"
    kubectl exec "${UI_POD}" -n registry -- cat /etc/nginx/conf.d/default.conf | head -20 || echo "Could not read nginx config"
fi


# #!/usr/bin/env bash

# echo "=== Registry UI Debug Script ==="
# echo

# echo "1. Checking pods in registry namespace:"
# kubectl get pods -n registry -o wide
# echo

# echo "2. Checking services in registry namespace:"
# kubectl get svc -n registry -o wide
# echo

# echo "3. Checking registry UI pod logs:"
# UI_POD=$(kubectl get pods -n registry -l app=registry-ui -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
# if [ -n "${UI_POD}" ]; then
#     echo "UI Pod: ${UI_POD}"
#     kubectl logs "${UI_POD}" -n registry --tail=20
# else
#     echo "No UI pod found with label app=registry-ui"
#     echo "Checking for other possible UI pods:"
#     kubectl get pods -n registry | grep -i ui || echo "No UI pods found"
# fi
# echo

# echo "4. Testing registry connectivity from within cluster:"
# REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
# if [ -n "${REGISTRY_POD}" ]; then
#     echo "Testing registry health endpoint:"
#     kubectl exec "${REGISTRY_POD}" -n registry -- wget -qO- http://localhost:5000/v2/ 2>/dev/null || echo "Registry health check failed"
    
#     echo "Testing service DNS resolution:"
#     kubectl exec "${REGISTRY_POD}" -n registry -- nslookup registry.registry.svc.cluster.local 2>/dev/null || echo "DNS resolution failed"
# else
#     echo "No registry pod found"
# fi
# echo

# echo "5. Checking registry UI environment variables:"
# if [ -n "${UI_POD}" ]; then
#     echo "Environment variables in UI pod:"
#     kubectl exec "${UI_POD}" -n registry -- env | grep -E "(REGISTRY|NGINX)" || echo "No registry-related env vars found"
# fi
# echo

# echo "6. Testing external access:"
# NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}' 2>/dev/null || echo "127.0.0.1")
# echo "Node IP: ${NODE_IP}"
# echo "Testing registry access: curl -I http://${NODE_IP}:31500/v2/"
# curl -I "http://${NODE_IP}:31500/v2/" 2>/dev/null || echo "Registry not accessible externally"
# echo

# echo "7. Registry configuration check:"
# if [ -n "${REGISTRY_POD}" ]; then
#     echo "Registry environment variables:"
#     kubectl exec "${REGISTRY_POD}" -n registry -- env | grep REGISTRY | head -10
# fi