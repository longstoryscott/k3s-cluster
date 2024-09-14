#! /bin/bash

VALUES=$1

SECRETS_DIR="${HOME}/.nc/secrets"

mkdir -p "${SECRETS_DIR}"

if [ ! -d "${SECRETS_DIR}" ]; then
    touch "${SECRETS_DIR}/nextcloud"
    touch "${SECRETS_DIR}/nextcloud-token"
    echo "enter nextcloud password"
    read -s -r pw
    echo "$pw" >"${SECRETS_DIR}/nextcloud"
    openssl rand -hex 10 | base64 >"${SECRETS_DIR}/nextcloud-token"
fi

kubectl create secret generic nextcloud-db \
    -n nextcloud \
    --from-file=db-password="${SECRETS_DIR}/nextcloud" \
    --from-literal=db-username=lsm \
    --save-config \
    --dry-run=client \
    --show-managed-fields=true \
    -o yaml | kubectl apply -f -

kubectl create secret generic nextcloud \
    -n nextcloud \
    --from-file=nextcloud-password="${SECRETS_DIR}/nextcloud" \
    --from-literal=nextcloud-username=lsm \
    --from-file=nextcloud-token="${SECRETS_DIR}/nextcloud-token" \
    --save-config \
    --dry-run=client \
    --show-managed-fields=true \
    -o yaml | kubectl apply -f -

sudo chown "$USER" "${SECRETS_DIR}" -R
sudo chmod 655 "${SECRETS_DIR}" -R

POSTGRES_PW=$(kubectl get secret --namespace nextcloud nextcloud-db -o jsonpath="{.data.db-password}" | base64 -d)
echo "POSTGRES_PW: ${POSTGRES_PW}"

helm repo add nextcloud https://nextcloud.github.io/helm/
helm repo update

if [ "${2}" = "template" ]; then
    helm template nextcloud nextcloud/nextcloud \
        --create-namespace -n nextcloud \
        --set postgresql.global.postgresql.auth.password="${POSTGRES_PW}" \
        --values "${VALUES}" >./manifest.yaml
else
    helm upgrade --install nextcloud nextcloud/nextcloud \
        --create-namespace -n nextcloud \
        --set postgresql.global.postgresql.auth.password="${POSTGRES_PW}" \
        --set externalDatabase.password="${POSTGRES_PW}" \
        --values "${VALUES}"

    kubectl wait --timeout=5m -n nextcloud deployment/nextcloud --for=condition=Available
fi
