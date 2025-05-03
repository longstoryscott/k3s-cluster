#!/bin/bash

# Script to import useful Grafana dashboards for logs and metrics visualization

# Make sure jq is installed
if ! command -v jq &>/dev/null; then
  echo "Error: jq is required but not installed. Please install jq first."
  exit 1
fi

# Start port-forwarding to grafana
echo "Starting port-forwarding to Grafana..."
kubectl port-forward -n monitoring svc/grafana 3000:80 &
FORWARD_PID=$!

# Wait for port-forwarding to establish
echo "Waiting for port-forwarding to establish..."
sleep 5

# Check if Grafana is accessible
if ! curl -s http://localhost:3000 >/dev/null; then
  echo "Error: Cannot access Grafana. Check if port-forwarding is working."
  kill $FORWARD_PID
  exit 1
fi

# Login to Grafana and get cookie (default credentials: admin/6u!tar00!QAZ)
echo "Logging into Grafana..."
COOKIE=$(curl -s -c - -X POST -H "Content-Type: application/json" -d '{"user":"admin","password":"6u!tar00!QAZ"}' http://localhost:3000/login | grep grafana_session | awk '{print $7}')

if [ -z "$COOKIE" ]; then
  echo "Error: Failed to login to Grafana."
  kill $FORWARD_PID
  exit 1
fi

# Function to import dashboard
import_dashboard() {
  DASHBOARD_ID=$1
  DASHBOARD_NAME=$2
  DATASOURCE=$3

  echo "Importing $DASHBOARD_NAME dashboard..."

  # Download dashboard JSON from grafana.com
  DASHBOARD_JSON=$(curl -s https://grafana.com/api/dashboards/$DASHBOARD_ID/revisions/latest/download)

  # Replace datasource if specified
  if [ ! -z "$DATASOURCE" ]; then
    DASHBOARD_JSON=$(echo $DASHBOARD_JSON | jq '.panels[].datasource = {"type": "'"$DATASOURCE"'", "uid": "'"$DATASOURCE"'"}' 2>/dev/null || echo $DASHBOARD_JSON)
    DASHBOARD_JSON=$(echo $DASHBOARD_JSON | jq '.templating.list[] |= if .type == "datasource" then .current = {"value": "'"$DATASOURCE"'", "selected": true} else . end' 2>/dev/null || echo $DASHBOARD_JSON)
  fi

  # Prepare import payload
  IMPORT_JSON="{\"dashboard\":$DASHBOARD_JSON,\"overwrite\":true}"

  # Import dashboard to Grafana
  RESULT=$(curl -s -b "grafana_session=$COOKIE" -X POST -H "Content-Type: application/json" -d "$IMPORT_JSON" http://localhost:3000/api/dashboards/db)

  if echo $RESULT | grep -q "success"; then
    echo "Successfully imported $DASHBOARD_NAME dashboard"
  else
    echo "Failed to import $DASHBOARD_NAME dashboard: $RESULT"
  fi
}

echo "Importing dashboards..."

# Loki dashboards
import_dashboard "12019" "Loki Dashboard" "Loki"
import_dashboard "15141" "Kubernetes Pod Logs" "Loki"
import_dashboard "13639" "Logs / App" "Loki"

# Kubernetes dashboards
import_dashboard "10856" "K8s / Node Exporter" "Prometheus"
import_dashboard "8588" "Kubernetes Deployment Statefulset Daemonset metrics" "Prometheus"
import_dashboard "11074" "Node Exporter for Prometheus Dashboard" "Prometheus"
import_dashboard "8685" "K8s Cluster Summary" "Prometheus"

echo "All dashboards imported successfully!"

# Kill the port-forwarding process
kill $FORWARD_PID

echo "You can now access Grafana dashboards at: http://localhost:3000"
echo "Use 'kubectl port-forward -n monitoring svc/grafana 3000:80' to access Grafana"
echo "Default credentials: admin/6u!tar00!QAZ"
