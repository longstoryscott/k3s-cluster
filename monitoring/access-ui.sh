#!/bin/bash

# Script to access monitoring UIs via port forwarding

case "$1" in
"grafana")
  echo "Starting port forwarding for Grafana..."
  echo "Access Grafana at http://localhost:3000"
  echo "Default credentials: admin/6u!tar00!QAZ"
  kubectl port-forward -n monitoring svc/grafana 3000:80
  ;;

"prometheus")
  echo "Starting port forwarding for Prometheus..."
  echo "Access Prometheus at http://localhost:9090"
  kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
  ;;

"alertmanager")
  echo "Starting port forwarding for Alertmanager..."
  echo "Access Alertmanager at http://localhost:9093"
  kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-alertmanager 9093:9093
  ;;

"loki")
  echo "Starting port forwarding for Loki..."
  echo "Access Loki at http://localhost:3100"
  kubectl port-forward -n monitoring svc/loki-stack 3100:3100
  ;;

"import-dashboards")
  echo "Running dashboard import script..."
  $(dirname "$0")/import-dashboards.sh
  ;;

*)
  echo "Usage: $0 [grafana|prometheus|alertmanager|loki|import-dashboards]"
  echo "  grafana            - Access Grafana UI at http://localhost:3000"
  echo "  prometheus         - Access Prometheus UI at http://localhost:9090"
  echo "  alertmanager       - Access Alertmanager UI at http://localhost:9093"
  echo "  loki               - Access Loki UI at http://localhost:3100"
  echo "  import-dashboards  - Import Grafana dashboards for logs and metrics"
  ;;
esac
