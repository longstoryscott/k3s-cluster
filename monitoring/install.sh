#!/bin/bash

# Create monitoring namespace if it doesn't exist
kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -

# Add the required Helm repositories
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Apply the Grafana configuration
kubectl apply -f "${1}/grafana-config.yaml"

# Install kube-prometheus-stack (includes Prometheus, AlertManager) but disable its Grafana
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --values "${1}/prometheus-values.yaml" \
  --set grafana.enabled=false \
  --timeout 15m

# Install Loki for log aggregation
helm upgrade --install loki-stack grafana/loki-stack \
  --namespace monitoring \
  --set grafana.enabled=false,prometheus.enabled=false,loki.persistence.enabled=true,loki.persistence.size=10Gi \
  --timeout 10m

# Install Promtail to collect logs and ship to Loki
helm upgrade --install promtail grafana/promtail \
  --namespace monitoring \
  --values "${1}/promtail-values.yaml" \
  --version 6.15.0 \
  --timeout 5m

# Update the standalone Grafana with our custom configuration
helm upgrade --install grafana grafana/grafana \
  --namespace monitoring \
  --values "${1}/grafana-values.yaml" \
  --timeout 5m

# Apply Loki data source
kubectl apply -f "${1}/loki-datasource.yaml"

kubectl apply -f "${1}/referencegrant.yaml"

# Show service endpoint information
echo
echo "Monitoring stack has been installed!"
echo
echo "Grafana is accessible via the NGINX Gateway at:"
echo "http://metrics.longstorymedia.com"
echo "Default credentials: admin/6u!tar00!QAZ"
echo
echo "Prometheus is accessible via the NGINX Gateway at:"
echo "http://metrics.longstorymedia.com/prometheus"
echo
echo "Loki is accessible via the NGINX Gateway at:"
echo "http://metrics.longstorymedia.com/loki"
