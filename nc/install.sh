#!/bin/bash

set -e

source "${1}/../helpers.sh"

# Create namespace
kubectl apply -f "${1}/namespace.yaml"

kubectl create secret generic nextcloud-token \
  -n nextcloud \
  "$(gen_secret nct)" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic nextcloud-db-pw \
  -n nextcloud \
  --from-file=psqlpw="${1}/../psql/.secrets/psqlpw" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic nextcloud-pw \
  -n nextcloud \
  --from-file=ncpw="${1}/.secrets/ncpw" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secrets created"
NC_DB_PW=$(kubectl get secret nextcloud-db-pw \
  -n nextcloud -o jsonpath="{.data.psqlpw}")
echo "DB password: $(echo $NC_DB_PW | base64 --decode)"
NC_PASSWORD=$(kubectl get secret nextcloud-pw \
  -n nextcloud -o jsonpath="{.data.ncpw}")
echo "Nextcloud password: $(echo $NC_PASSWORD | base64 --decode)"
NC_TOKEN=$(kubectl get secret nextcloud-token \
  -n nextcloud -o jsonpath="{.data.nct}")
echo "Nextcloud token: $(echo $NC_TOKEN | base64 --decode)"
NC_USERNAME=$(echo "lsm" | base64 -i -)
echo "Nextcloud username: ${NC_USERNAME}"
# Create secrets.yaml with the right values
cat >"${1}/.secrets/secrets.yaml" <<EOF_SECRET
apiVersion: v1
kind: Secret
metadata:
  name: nextcloud-db
  namespace: nextcloud
type: Opaque
data:
  db-username: ${NC_USERNAME}
  db-password: ${NC_DB_PW}
---
apiVersion: v1
kind: Secret
metadata:
  name: nextcloud
  namespace: nextcloud
type: Opaque
data:
  nextcloud-username: ${NC_USERNAME}
  nextcloud-password: ${NC_PASSWORD}
  nextcloud-token: ${NC_TOKEN}
EOF_SECRET

# Apply all the manifests in order
kubectl apply -f "${1}/.secrets/secrets.yaml"
# kubectl apply -f "${1}/03-postgresql.yaml"
kubectl apply -f "${1}/configmap.yaml"
kubectl apply -f "${1}/pvc.yaml"
kubectl apply -f "${1}/deployment.yaml"
kubectl apply -f "${1}/service.yaml"
kubectl apply -f "${1}/referencegrant.yaml"

# Wait for deployment to be available
echo "Waiting for Nextcloud deployment to be ready..."
kubectl wait --timeout=5m -n nextcloud deployment/nextcloud --for=condition=Available

echo "Nextcloud installation complete!"
