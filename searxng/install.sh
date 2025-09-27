#!/usr/bin/env bash
set -e

source "$(dirname "$0")/../helpers.sh"

kubectl create namespace searxng || true

# Create secret for SearXNG
kubectl create secret generic secrets \
  -n searxng \
  "$(gen_secret searxng_secret_key)" \
  --dry-run=client -o yaml | kubectl apply -f - --wait=true

# Apply SearXNG manifests
kubectl apply -n searxng -f "$(dirname "$0")/serviceaccount.yaml" --wait=true
kubectl apply -n searxng -f "$(dirname "$0")/pvc.yaml" --wait=true
kubectl apply -n searxng -f "$(dirname "$0")/deployment.yaml" --wait=true
kubectl apply -n searxng -f "$(dirname "$0")/service.yaml" --wait=true
kubectl apply -n searxng -f "$(dirname "$0")/referencegrant.yaml" --wait=true

# Check SearXNG pod status
echo "Waiting for SearXNG Deployment to start..."
kubectl rollout status deployment/searxng -n searxng --timeout=2m || true

# Display pod and PVC status
echo "Current pod status:"
kubectl get pods -n searxng
echo "Service status:"
kubectl get svc -n searxng

# Get node IPs for access information
NODE_IPS=$(kubectl get nodes -o jsonpath='{.items[*].status.addresses[?(@.type=="InternalIP")].address}')

echo ""
echo "SearXNG is now available at:"
echo "- NodePort: http://<node-ip>:32348"
echo "  (Where <node-ip> can be any of: $NODE_IPS)"
echo "- You can also access it through the LoadBalancer External-IPs listed above"
echo ""
echo "SearXNG is a privacy-respecting metasearch engine ready for your use."
