#!/usr/bin/env bash
set -e

source "${1}/../helpers.sh"

CWD=$1
PATCH=$2
REGISTRY_HOME="${HOME}/.registry"
CERTS_DIR="${REGISTRY_HOME}/certs"
AUTH_DIR="${REGISTRY_HOME}/auth"
HTPASSWD_FILE="${AUTH_DIR}/htpasswd"
SECRETS_DIR="${CWD}/.secrets"
mkdir -p "${SECRETS_DIR}"

echo "Setting up private container registry with TLS termination..."

# Create namespace for the registry
kubectl create namespace registry || true

# Create necessary directories
mkdir -p "${REGISTRY_HOME}" "${CERTS_DIR}" "${AUTH_DIR}" || true

# Generate user/password paths
USER_SECRET_FILE="${SECRETS_DIR}/registryuser"
PW_SECRET_FILE="${SECRETS_DIR}/registrypw"

# Create secrets if they don't exist
[[ -f "$USER_SECRET_FILE" ]] || openssl rand -hex 8 >"$USER_SECRET_FILE"
[[ -f "$PW_SECRET_FILE" ]] || openssl rand -hex 12 >"$PW_SECRET_FILE"

# Read the secrets
REGISTRY_USER=$(cat "$USER_SECRET_FILE")
REGISTRY_PW=$(cat "$PW_SECRET_FILE")

# Get the node IP address
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}')
REGISTRY_PORT="31500"
REGISTRY_URL="${NODE_IP}:${REGISTRY_PORT}"

echo "Registry will be accessible at: https://${REGISTRY_URL}"

# Generate self-signed certificates for TLS if they don't exist
if [ ! -f "${CERTS_DIR}/tls.key" ] || [ ! -f "${CERTS_DIR}/tls.crt" ]; then
    echo "Generating self-signed certificates for TLS..."

    # Create an OpenSSL config to include IP address in SAN
    cat >"${CERTS_DIR}/openssl.cnf" <<EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = ${NODE_IP}

[v3_req]
subjectAltName = @alt_names

[alt_names]
IP.1 = ${NODE_IP}
IP.2 = 127.0.0.1
EOF

    # Create a certificate with the IP in both CN and SAN
    openssl req -x509 \
        -newkey rsa:4096 \
        -days 365 -nodes -sha256 \
        -keyout "${CERTS_DIR}/tls.key" \
        -out "${CERTS_DIR}/tls.crt" \
        -config "${CERTS_DIR}/openssl.cnf" \
        -extensions v3_req

    echo "Certificates generated successfully."
fi

# Create Kubernetes secret for TLS certificates
kubectl create secret tls registry-certs-secret \
    -n registry \
    --cert="${CERTS_DIR}/tls.crt" \
    --key="${CERTS_DIR}/tls.key" \
    --dry-run=client -o yaml | kubectl apply -f - --wait=true

# Set up htpasswd file for registry authentication
echo "Creating authentication file..."
# Create htpasswd file
docker run --rm \
    --entrypoint htpasswd \
    httpd:2 -Bbn "${REGISTRY_USER}" "${REGISTRY_PW}" >"${HTPASSWD_FILE}"

# Create auth secret from htpasswd file
kubectl create secret generic registry-auth \
    -n registry \
    --from-file=htpasswd="${HTPASSWD_FILE}" \
    --dry-run=client -o yaml | kubectl apply -f - --wait=true

echo "Authentication configured successfully."

# First cleanup any existing resources to ensure a fresh start
echo "Cleaning up any existing registry resources..."
kubectl delete deployment/registry -n registry --ignore-not-found=true

# Apply Kubernetes resources
echo "Deploying registry resources..."
kubectl apply -n registry -f "${CWD}/pvc.yaml" --wait=true
kubectl apply -n registry -f "${CWD}/serviceaccount.yaml" --wait=true
kubectl apply -n registry -f "${CWD}/deployment.yaml" --wait=true
kubectl apply -n registry -f "${CWD}/service.yaml" --wait=true

echo "Waiting for registry deployment to start..."
kubectl rollout status deployment/registry -n registry --timeout=2m || true

# Check pod status and add troubleshooting info
echo "Current pod status:"
kubectl get pods -n registry
echo "Pod description:"
REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "${REGISTRY_POD}" ]; then
    kubectl describe pod "${REGISTRY_POD}" -n registry | grep -A 10 "Events:"
else
    echo "No registry pod found."
fi
echo "PVC status:"
kubectl get pvc -n registry

echo "Registry setup completed - checking registry availability..."
# Wait for pod to be ready and try to connect to the registry
if [ -n "${REGISTRY_POD}" ]; then
    timeout=60
    counter=0
    ready=false
    while [ $counter -lt $timeout ] && [ "$ready" = false ]; do
        if kubectl get pod "${REGISTRY_POD}" -n registry -o jsonpath='{.status.phase}' 2>/dev/null | grep -q "Running"; then
            ready=true
            echo "Registry is running!"
        else
            counter=$((counter + 5))
            echo "Waiting for registry to be ready... ($counter/$timeout seconds)"
            sleep 5
        fi
    done

    if [ "$ready" = false ]; then
        echo "Registry failed to become ready within timeout period. Check logs for more details:"
        kubectl logs "${REGISTRY_POD}" -n registry 2>/dev/null || echo "Could not retrieve logs"
        exit 1
    else
        echo "Registry is ready!"
        echo "Setting up Docker certificates for secure registry connection..."

        # Detect operating system
        OS="$(uname -s)"
        case "${OS}" in
        Darwin*) IS_MAC=true ;;
        Linux*) IS_MAC=false ;;
        *)
            echo "Unsupported operating system: ${OS}"
            exit 1
            ;;
        esac

        # Create Docker certs directory
        DOCKER_CERTS_DIR="${HOME}/.docker/certs.d/${REGISTRY_URL}"
        mkdir -p "${DOCKER_CERTS_DIR}" || true

        # Copy the registry certificate to Docker certs directory
        cp "${CERTS_DIR}/tls.crt" "${DOCKER_CERTS_DIR}/ca.crt"
        echo "Certificate copied to Docker certs directory: ${DOCKER_CERTS_DIR}"

        # For macOS, we need to add the certificate to the keychain
        if [ "${IS_MAC}" = true ]; then
            echo "Adding certificate to macOS keychain..."
            security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "${CERTS_DIR}/tls.crt" ||
                echo "⚠️ Failed to add certificate to system keychain. You may need to run: sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ${CERTS_DIR}/tls.crt"

            echo "⚠️ On macOS, you may need to restart Docker Desktop for the certificates to take effect."
            echo "You can do this by running: killall Docker && open -a Docker"
        fi

        echo ""
        echo "================================================"
        echo "Registry Setup Summary"
        echo "================================================"
        echo "Registry address: https://${REGISTRY_URL}"
        echo "Registry credentials:"
        echo "  Username: ${REGISTRY_USER}"
        echo "  Password: [stored in ${PW_SECRET_FILE}]"
        echo ""
        echo "Docker login command:"
        echo "  echo \"${REGISTRY_PW}\" | docker login ${REGISTRY_URL} -u ${REGISTRY_USER} --password-stdin"
        echo ""
        echo "To push an image to your registry:"
        echo "  docker tag myimage:latest ${REGISTRY_URL}/myimage:latest"
        echo "  docker push ${REGISTRY_URL}/myimage:latest"
        echo ""
        echo "To pull an image from your registry:"
        echo "  docker pull ${REGISTRY_URL}/myimage:latest"
        echo "================================================"

        # Try to login automatically
        echo "Attempting automatic login to Docker registry..."
        # Create a temp file for the password to avoid it showing in process list
        password_file=$(mktemp)
        echo "${REGISTRY_PW}" >"${password_file}"

        # Attempt login
        if cat "${password_file}" | docker login "https://${REGISTRY_URL}" -u "${REGISTRY_USER}" --password-stdin; then
            echo "✅ Successfully logged in to registry at ${REGISTRY_URL}"
            echo "Registry is ready to use!"
        else
            echo "❌ Failed to log in automatically. Please try manually:"
            echo "echo \"${REGISTRY_PW}\" | docker login ${REGISTRY_URL} -u ${REGISTRY_USER} --password-stdin"
        fi

        # Clean up the temp file
        rm -f "${password_file}"
    fi
else
    echo "Registry pod not found. Deployment may have failed."
    exit 1
fi
