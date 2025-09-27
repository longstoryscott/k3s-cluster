#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Make sure registry namespace exists
if ! kubectl get namespace registry >/dev/null 2>&1; then
  echo "Creating registry namespace..."
  kubectl create namespace registry
fi

# Check if registry deployment exists
if ! kubectl get deployment registry -n registry >/dev/null 2>&1; then
  echo "Error: Registry deployment not found. Please install the main registry first with 'make registry'"
  exit 1
fi

# Deploy the registry UI (it will work with the existing registry even without CORS)
echo "Deploying registry UI..."
kubectl apply -f "${SCRIPT_DIR}/deployment.yaml"
kubectl apply -f "${SCRIPT_DIR}/service.yaml"

# Wait for the registry-ui pod to be ready
echo "Waiting for registry-ui pod to be ready..."
kubectl wait --for=condition=ready pod -l app=registry-ui -n registry --timeout=90s

# Get service info
REGISTRY_UI_PORT=$(kubectl get svc registry-ui -n registry -o jsonpath='{.spec.ports[0].nodePort}')
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')

echo "======================================================================"
echo "Registry UI deployed successfully!"
echo "You can access it at: http://${NODE_IP}:${REGISTRY_UI_PORT}"
echo "======================================================================"
echo "Registry username and password are required for authentication."
echo "These are the same credentials you use for the registry itself."
echo ""
echo "Note: If you experience CORS issues in the browser, you may need to"
echo "update the registry configuration to include CORS headers."
echo "======================================================================"
