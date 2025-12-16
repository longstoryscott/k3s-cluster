#!/bin/bash

# Script to idempotently deploy NGINX Gateway Fabric using manifests
# This script can be run multiple times safely

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Installing NGINX Gateway Fabric with LoadBalancer..."

# Apply Gateway API CRDs (idempotent)
echo "📦 Applying Gateway API CRDs..."
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

# Apply NGINX Gateway Fabric CRDs (handle conflicts)  
echo "📦 Applying NGINX Gateway Fabric CRDs..."
kubectl apply -f https://raw.githubusercontent.com/nginxinc/nginx-gateway-fabric/v2.1.4/deploy/crds.yaml --force-conflicts=true --server-side=true

# Apply the main deployment (idempotent)
echo "🔧 Applying NGINX Gateway Fabric deployment..."
# Delete the cert generator job if it exists (jobs are not idempotent)
kubectl delete job nginx-gateway-cert-generator -n nginx-gateway --ignore-not-found=true
kubectl apply -k "${SCRIPT_DIR}/manifests"

# Wait for the deployment to be ready
echo "⏳ Waiting for NGINX Gateway to be ready..."
kubectl rollout status deployment/nginx-gateway -n nginx-gateway --timeout=300s

# Apply the Gateway and HTTPRoutes configuration
echo "🌐 Applying Gateway and HTTPRoute configurations..."
kubectl apply -f "${SCRIPT_DIR}/routes.yaml" --wait=true

# Wait for Gateway to be programmed
echo "⏳ Waiting for Gateway to be programmed..."
kubectl wait --for=condition=Programmed gateway/lsm-gateway -n nginx-gateway --timeout=60s

# Get the LoadBalancer service information
echo "✅ NGINX Gateway Fabric deployment complete!"
echo ""

GATEWAY_IP=$(kubectl get svc -n nginx-gateway lsm-gateway-nginx -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
if [ -z "$GATEWAY_IP" ]; then
  echo "⏳ LoadBalancer IP not yet assigned. Checking service status..."
  kubectl get svc -n nginx-gateway lsm-gateway-nginx
  echo ""
  echo "💡 Run 'kubectl get svc -n nginx-gateway lsm-gateway-nginx' to check for external IP assignment"
else
  echo "🌍 LoadBalancer External IP: $GATEWAY_IP"
  echo ""
  echo "🔗 Service Access URLs:"
  echo "   • NextCloud:      http://$GATEWAY_IP:3000"
  echo "   • Auth Service:   http://$GATEWAY_IP:9091"
  echo "   • Grafana:        http://$GATEWAY_IP:3001"
  echo "   • Prometheus:     http://$GATEWAY_IP:9090"
  echo "   • User Manager:   http://$GATEWAY_IP:3333"
  echo "   • Registry:       http://$GATEWAY_IP:5000"
  echo "   • Registry UI:    http://$GATEWAY_IP:8085"
fi

echo ""
echo "🎯 Gateway Status:"
kubectl get gateway lsm-gateway -n nginx-gateway -o wide

echo ""
echo "📊 HTTPRoute Status:"
kubectl get httproute -n nginx-gateway

echo ""
echo "✨ Deployment completed successfully!"
