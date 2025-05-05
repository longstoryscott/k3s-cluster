#!/usr/bin/env bash
set -e

# Consolidated Registry Management Script
# Handles certificates, authentication, and Docker configuration

# Configuration
REGISTRY_HOME="${HOME}/.registry"
CERTS_DIR="${REGISTRY_HOME}/certs"
AUTH_DIR="${REGISTRY_HOME}/auth"
HTPASSWD_FILE="${AUTH_DIR}/htpasswd"
SECRETS_DIR="$(dirname "$0")/.secrets"
CERT_DIR="/tmp/registry-certs"
DOCKER_CONFIG_DIR="${HOME}/.docker"
DOCKER_CONFIG_FILE="${DOCKER_CONFIG_DIR}/daemon.json"

# Get the node IP address
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}' 2>/dev/null || echo "127.0.0.1")
REGISTRY_PORT="31500"
REGISTRY_DNS="registry.local"
REGISTRY_URL="${NODE_IP}:${REGISTRY_PORT}"

function print_usage() {
  cat <<EOF
Registry Management Utility

Usage:
  ./registry-mgmt.sh [command]

Available Commands:
  install                            - Install registry with default settings
  gen-certs                          - Generate new TLS certificates
  trust-cert                         - Trust certificate on macOS
  configure-docker                   - Configure Docker for this registry
  add-user    <username> <password>  - Add a new user to the registry
  del-user    <username>             - Delete a user from the registry
  change-pw   <username> <password>  - Change a user's password
  list-users                         - List all users in the registry
  restart                            - Restart the registry pod

Examples:
  ./registry-mgmt.sh install
  ./registry-mgmt.sh add-user developer password123
  ./registry-mgmt.sh trust-cert

EOF
}

function init_directories() {
  mkdir -p "${REGISTRY_HOME}" "${CERTS_DIR}" "${AUTH_DIR}" "${SECRETS_DIR}" || true
  mkdir -p "${DOCKER_CONFIG_DIR}/certs.d/${REGISTRY_URL}" || true
  touch "${HTPASSWD_FILE}" || true
  chmod 600 "${HTPASSWD_FILE}" || true
}

function install() {
  echo "Setting up private container registry..."

  # Create namespace for the registry
  kubectl create namespace registry || true

  # Initialize directories
  init_directories

  # Generate credentials if they don't exist
  USER_SECRET_FILE="${SECRETS_DIR}/registryuser"
  PW_SECRET_FILE="${SECRETS_DIR}/registrypw"

  [[ -f "$USER_SECRET_FILE" ]] || openssl rand -hex 8 >"$USER_SECRET_FILE"
  [[ -f "$PW_SECRET_FILE" ]] || openssl rand -hex 12 >"$PW_SECRET_FILE"

  # Read the secrets
  REGISTRY_USER=$(cat "$USER_SECRET_FILE")
  REGISTRY_PW=$(cat "$PW_SECRET_FILE")

  # Generate certificates if they don't exist
  if [ ! -f "${CERTS_DIR}/tls.key" ] || [ ! -f "${CERTS_DIR}/tls.crt" ]; then
    generate_certificates
  fi

  # Create htpasswd file if it doesn't exist
  if [ ! -s "${HTPASSWD_FILE}" ]; then
    echo "Creating initial authentication with random credentials..."
    docker run --rm \
      --entrypoint htpasswd \
      httpd:2 -Bbn "${REGISTRY_USER}" "${REGISTRY_PW}" >"${HTPASSWD_FILE}"
    echo "Created user ${REGISTRY_USER}"
  fi

  # Create Kubernetes secrets
  create_registry_secrets

  # Apply deployment
  echo "Deploying registry..."
  kubectl apply -f "$(dirname "$0")/deployment.yaml" -n registry
  kubectl apply -f "$(dirname "$0")/service.yaml" -n registry
  kubectl apply -f "$(dirname "$0")/pvc.yaml" -n registry
  kubectl apply -f "$(dirname "$0")/serviceaccount.yaml" -n registry

  if [ -f "$(dirname "$0")/referencegrant.yaml" ]; then
    kubectl apply -f "$(dirname "$0")/referencegrant.yaml" -n registry
  fi

  echo "Registry deployment complete!"
  echo "Registry will be accessible at: https://${REGISTRY_URL}"
  echo "Username: ${REGISTRY_USER}"
  echo "Password: ${REGISTRY_PW}"
  echo
  echo "You may need to run:"
  echo "  $0 trust-cert        # To trust the certificate on macOS"
  echo "  $0 configure-docker  # To configure Docker for this registry"
}

function generate_certificates() {
  echo "Generating certificates with proper SAN fields..."

  # Clean and create certificate directory
  rm -rf "${CERT_DIR}"
  mkdir -p "${CERT_DIR}"

  # Create OpenSSL config with proper SAN entries
  cat >"${CERT_DIR}/openssl.cnf" <<EOF
[ req ]
default_bits = 2048
prompt = no
default_md = sha256
req_extensions = req_ext
distinguished_name = dn

[ dn ]
CN = ${REGISTRY_DNS}

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
IP.1 = 127.0.0.1
IP.2 = ${NODE_IP}
DNS.1 = localhost
DNS.2 = ${REGISTRY_DNS}
DNS.3 = registry
DNS.4 = registry.local
DNS.5 = ${REGISTRY_URL}
DNS.6 = ${NODE_IP}
DNS.7 = registry.local:5000
DNS.8 = ${REGISTRY_DNS}:5000
DNS.9 = ${NODE_IP}:5000
EOF

  echo "Generating private key..."
  openssl genrsa -out "${CERT_DIR}/tls.key" 4096

  echo "Generating certificate signing request with SAN..."
  openssl req -new -key "${CERT_DIR}/tls.key" -out "${CERT_DIR}/tls.csr" -config "${CERT_DIR}/openssl.cnf"

  echo "Creating self-signed certificate valid for 10 years..."
  openssl x509 -req -days 3650 -signkey "${CERT_DIR}/tls.key" -in "${CERT_DIR}/tls.csr" \
    -out "${CERT_DIR}/tls.crt" -extfile "${CERT_DIR}/openssl.cnf" -extensions req_ext

  # Verify the certificate SAN fields
  echo "Verifying certificate..."
  openssl x509 -in "${CERT_DIR}/tls.crt" -text -noout | grep -A1 "Subject Alternative Name"

  # Copy certificates to registry directory
  cp "${CERT_DIR}/tls.crt" "${CERTS_DIR}/tls.crt"
  cp "${CERT_DIR}/tls.key" "${CERTS_DIR}/tls.key"
  cp "${CERT_DIR}/tls.crt" "${DOCKER_CONFIG_DIR}/certs.d/${REGISTRY_URL}/ca.crt"
  cp "${CERT_DIR}/tls.crt" "${HOME}/Desktop/registry-cert.crt" 2>/dev/null || true

  echo "✅ Certificate generated successfully!"
}

function create_registry_secrets() {
  # Create Kubernetes secrets for registry certificates
  kubectl create secret tls registry-certs-secret -n registry \
    --cert="${CERTS_DIR}/tls.crt" \
    --key="${CERTS_DIR}/tls.key" \
    --dry-run=client -o yaml | kubectl apply -f -

  # Create Kubernetes secret for registry auth
  kubectl create secret generic registry-auth \
    -n registry \
    --from-file="${HTPASSWD_FILE}" \
    --dry-run=client -o yaml | kubectl apply -f -
}

function trust_cert() {
  # Function to help trust the certificate on macOS
  if [ ! -f "${CERTS_DIR}/tls.crt" ]; then
    echo "❌ Certificate file not found at ${CERTS_DIR}/tls.crt"
    echo "Please run gen-certs first."
    exit 1
  fi

  # Ensure Docker cert directory exists and has the certificate
  mkdir -p "${DOCKER_CONFIG_DIR}/certs.d/${REGISTRY_URL}"
  cp "${CERTS_DIR}/tls.crt" "${DOCKER_CONFIG_DIR}/certs.d/${REGISTRY_URL}/ca.crt"
  echo "✅ Certificate copied to Docker certs directory"

  # On macOS, we need to use the keychain
  OS="$(uname -s)"
  if [ "${OS}" = "Darwin" ]; then
    echo "📱 macOS detected: Setting up system trust..."

    # Try to add to user's keychain
    echo "Adding certificate to user's login keychain..."
    if security add-trusted-cert -d -r trustRoot -k ~/Library/Keychains/login.keychain-db "${CERTS_DIR}/tls.crt"; then
      echo "✅ Certificate added to user keychain!"
    else
      echo "❌ Failed to add to user keychain."
    fi

    # Create a copy that can be manually installed
    DESKTOP_CERT="${HOME}/Desktop/registry-cert.crt"
    cp "${CERTS_DIR}/tls.crt" "${DESKTOP_CERT}"
    echo
    echo "⚠️ A copy of the certificate has been saved to ${DESKTOP_CERT}"
    echo "Please double-click this file in Finder and when prompted:"
    echo "1. Choose 'Add to System' (not login) keychain"
    echo "2. Click 'Always Trust' when the certificate opens in Keychain Access"
  fi

  echo
  echo "Remember to restart Docker completely (quit and reopen Docker Desktop)"
}

function configure_docker() {
  echo "Configuring Docker to use registry at ${REGISTRY_URL}..."

  # Create Docker config directory if it doesn't exist
  mkdir -p "${DOCKER_CONFIG_DIR}/certs.d/${REGISTRY_URL}" || true
  mkdir -p "${DOCKER_CONFIG_DIR}/certs.d/registry.local:5000" || true

  # Copy certificates to Docker certs directories
  cp "${CERTS_DIR}/tls.crt" "${DOCKER_CONFIG_DIR}/certs.d/${REGISTRY_URL}/ca.crt"
  cp "${CERTS_DIR}/tls.crt" "${DOCKER_CONFIG_DIR}/certs.d/registry.local:5000/ca.crt"

  # Create or update daemon.json
  if [ ! -f "${DOCKER_CONFIG_FILE}" ]; then
    # Create new file if it doesn't exist
    echo '{
  "insecure-registries": ["'${REGISTRY_URL}'", "registry.local:5000"]
}' >"${DOCKER_CONFIG_FILE}"
    echo "✅ Created new Docker daemon config file with insecure registry."
  else
    # File exists, check if our registry is already in it
    if grep -q "${REGISTRY_URL}" "${DOCKER_CONFIG_FILE}" && grep -q "registry.local:5000" "${DOCKER_CONFIG_FILE}"; then
      echo "✅ Registry already configured in Docker daemon."
    else
      # Make a backup of the existing file
      cp "${DOCKER_CONFIG_FILE}" "${DOCKER_CONFIG_FILE}.bak"
      echo "📁 Backed up existing daemon.json to daemon.json.bak"

      # Use temporary file to update the daemon.json
      TMP_FILE=$(mktemp)

      # Add both registry URLs to insecure-registries if needed
      if ! grep -q "${REGISTRY_URL}" "${DOCKER_CONFIG_FILE}"; then
        jq '.["insecure-registries"] += ["'${REGISTRY_URL}'"]' "${DOCKER_CONFIG_FILE}" >"${TMP_FILE}" || {
          echo "⚠️ Failed to update daemon.json with jq, using simpler approach"
          sed 's/\("insecure-registries": \[\)/\1"'${REGISTRY_URL}'", /' "${DOCKER_CONFIG_FILE}"
        }
        mv "${TMP_FILE}" "${DOCKER_CONFIG_FILE}"
      fi

      if ! grep -q "registry.local:5000" "${DOCKER_CONFIG_FILE}"; then
        jq '.["insecure-registries"] += ["registry.local:5000"]' "${DOCKER_CONFIG_FILE}" >"${TMP_FILE}" || {
          echo "⚠️ Failed to update daemon.json with jq, using simpler approach"
          sed 's/\("insecure-registries": \[\)/\1"registry.local:5000", /' "${DOCKER_CONFIG_FILE}"
        }
        mv "${TMP_FILE}" "${DOCKER_CONFIG_FILE}"
      fi
      echo "✅ Updated insecure-registries in daemon.json."
    fi
  fi

  echo "Docker daemon configuration updated."
  echo "Current daemon.json content:"
  cat "${DOCKER_CONFIG_FILE}"

  echo
  echo "IMPORTANT: Restart Docker for changes to take effect"
  echo "On macOS: Quit Docker Desktop from the menu bar and start it again"
}

function add_user() {
  if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Error: Username and password required"
    print_usage
    exit 1
  fi

  init_directories

  echo "Adding user $1..."
  docker run --rm \
    --entrypoint htpasswd \
    httpd:2 -Bbn "$1" "$2" >>"${HTPASSWD_FILE}"

  update_auth_secret
  echo "User $1 added successfully."
}

function delete_user() {
  if [ -z "$1" ]; then
    echo "Error: Username required"
    print_usage
    exit 1
  fi

  if [ ! -f "${HTPASSWD_FILE}" ]; then
    echo "Error: Authentication file does not exist"
    exit 1
  fi

  echo "Deleting user $1..."
  # Create a temporary file without the user's line
  grep -v "^$1:" "${HTPASSWD_FILE}" >"${HTPASSWD_FILE}.tmp" || true
  mv "${HTPASSWD_FILE}.tmp" "${HTPASSWD_FILE}"

  update_auth_secret
  echo "User $1 deleted successfully."
}

function change_password() {
  if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Error: Username and new password required"
    print_usage
    exit 1
  fi

  delete_user "$1"
  add_user "$1" "$2"
  echo "Password for $1 changed successfully."
}

function list_users() {
  if [ ! -f "${HTPASSWD_FILE}" ]; then
    echo "No users found. Authentication file does not exist."
    return
  fi

  echo "Registered users:"
  grep -o "^[^:]*" "${HTPASSWD_FILE}" | sort
}

function update_auth_secret() {
  kubectl create secret generic registry-auth \
    -n registry \
    --from-file="${HTPASSWD_FILE}" \
    --dry-run=client -o yaml | kubectl apply -f -

  restart_registry_pod
}

function restart_registry_pod() {
  # Restart the registry pod to apply changes
  REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  if [ -n "${REGISTRY_POD}" ]; then
    echo "Restarting registry pod to apply changes..."
    kubectl delete pod "${REGISTRY_POD}" -n registry
  else
    echo "Registry pod not found. Is the registry deployed?"
  fi
}

# Consolidate redundant functionality from other scripts into this one.
# Add modular functions for certificate management, Docker configuration, and user management.

function manage_certificates() {
  echo "Managing certificates..."
  # Combine logic from setup-certs.sh and trust-cert.sh here
  generate_certificates
  create_registry_secrets
  trust_cert
  restart_registry_pod
}

function manage_users() {
  echo "Managing users..."
  # Combine logic from accessmgmt.sh here
  case "$2" in
  add)
    add_user "$3" "$4"
    ;;
  delete)
    delete_user "$3"
    ;;
  change-pw)
    change_password "$3" "$4"
    ;;
  list)
    list_users
    ;;
  *)
    echo "Invalid user management command"
    print_usage
    ;;
  esac
}

# Main execution logic
init_directories

case "$1" in
install)
  install
  ;;
gen-certs)
  generate_certificates
  create_registry_secrets
  restart_registry_pod
  ;;
trust-cert)
  trust_cert
  ;;
configure-docker)
  configure_docker
  ;;
add-user)
  add_user "$2" "$3"
  ;;
del-user)
  delete_user "$2"
  ;;
change-pw)
  change_password "$2" "$3"
  ;;
list-users)
  list_users
  ;;
restart)
  restart_registry_pod
  ;;
manage-certs)
  manage_certificates
  ;;
manage-users)
  manage_users "$@"
  ;;
*)
  print_usage
  ;;
esac
