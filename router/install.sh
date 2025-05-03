#!/bin/bash

# Script to apply the monitoring routes configuration to your K3s cluster

echo "Applying updated Gateway configuration with monitoring routes..."

kubectl kustomize "https://github.com/nginx/nginx-gateway-fabric/config/crd/gateway-api/standard?ref=v1.6.2" | kubectl apply -f -

# Create the namespace if it doesn't exist
kubectl create namespace nginx-gateway || true

# Install or upgrade NGINX Gateway with our values
helm upgrade --install nginx-gateway oci://ghcr.io/nginxinc/charts/nginx-gateway-fabric --create-namespace -n nginx-gateway --values ${1}/values.yaml
kubectl wait --timeout=5m -n nginx-gateway deployment/nginx-gateway-nginx-gateway-fabric --for=condition=Available

# Apply the updated routes
kubectl apply -f ${1}/routes.yaml --wait=true
kubectl rollout status deployment/nginx-gateway-nginx-gateway-fabric -n nginx-gateway --timeout=2m || true

# Restart the NGINX Gateway (only if needed)
GATEWAY_POD=$(kubectl get pods -n nginx-gateway -l app.kubernetes.io/name=nginx-gateway-fabric -o jsonpath='{.items[0].metadata.name}')
if [ ! -z "$GATEWAY_POD" ]; then
  echo "Restarting NGINX Gateway pod: $GATEWAY_POD"
  kubectl delete pod $GATEWAY_POD -n nginx-gateway
  echo "Waiting for new pod to start..."
  sleep 5
  kubectl get pods -n nginx-gateway -l app.kubernetes.io/name=nginx-gateway-fabric
else
  echo "NGINX Gateway pod not found. Is it running in a different namespace?"
fi

echo "Monitoring UI routes have been configured."
echo ""
GATEWAY_IP=$(kubectl get svc -n nginx-gateway nginx-gateway-nginx-gateway-fabric -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -z "$GATEWAY_IP" ]; then
  GATEWAY_IP="<your-gateway-ip>"
fi

echo "You can now access your monitoring UIs at:"
echo "- Grafana:      http://$GATEWAY_IP:3001"
echo "- Prometheus:   http://$GATEWAY_IP:9090"
echo "- Alertmanager: http://$GATEWAY_IP:9093"
echo ""
echo "To get your Gateway IP address, run:"
echo "kubectl get svc -n nginx-gateway nginx-gateway-nginx-gateway-fabric"
