#!/usr/bin/env bash
set -e

# Simple Registry Management Script (HTTP only)

REGISTRY_HOME="${HOME}/.registry"
AUTH_DIR="${REGISTRY_HOME}/auth"
HTPASSWD_FILE="${AUTH_DIR}/htpasswd"
SECRETS_DIR="$(dirname "$0")/.secrets"
DOCKER_CONFIG_DIR="${HOME}/.docker"
DOCKER_CONFIG_FILE="${DOCKER_CONFIG_DIR}/daemon.json"
NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[0].address}' 2>/dev/null || echo "127.0.0.1")
REGISTRY_PORT="31500"
REGISTRY_DNS="registry.local"
REGISTRY_URL="${NODE_IP}:${REGISTRY_PORT}"

function print_usage() {
  cat <<EOF
Simple Registry Management Utility (HTTP only)

Usage:
  ./registry-mgmt.sh [command]

Available Commands:
  install-simple                     - Install registry with simplified settings (HTTP, no TLS)
  configure-docker-simple            - Configure Docker for simple HTTP registry
  add-user    <username> <password>  - Add a new user to the registry
  del-user    <username>             - Delete a user from the registry
  change-pw   <username> <password>  - Change a user's password
  list-users                         - List all users in the registry
  restart                            - Restart the registry pod

Examples:
  ./registry-mgmt.sh install-simple
  ./registry-mgmt.sh add-user developer password123
EOF
}

function init_directories() {
  mkdir -p "${REGISTRY_HOME}" "${AUTH_DIR}" "${SECRETS_DIR}" || true
  touch "${HTPASSWD_FILE}" || true
  chmod 600 "${HTPASSWD_FILE}" || true
}

function install_simple() {
  echo "Setting up simplified private container registry (HTTP mode)..."
  kubectl create namespace registry || true
  init_directories
  USER_SECRET_FILE="${SECRETS_DIR}/registryuser"
  PW_SECRET_FILE="${SECRETS_DIR}/registrypw"
  [[ -f "$USER_SECRET_FILE" ]] || openssl rand -hex 8 >"$USER_SECRET_FILE"
  [[ -f "$PW_SECRET_FILE" ]] || openssl rand -hex 12 >"$PW_SECRET_FILE"
  REGISTRY_USER=$(cat "$USER_SECRET_FILE")
  REGISTRY_PW=$(cat "$PW_SECRET_FILE")
  if [ ! -s "${HTPASSWD_FILE}" ]; then
    echo "Creating initial authentication with random credentials..."
    docker run --rm --entrypoint htpasswd httpd:2 -Bbn "${REGISTRY_USER}" "${REGISTRY_PW}" >"${HTPASSWD_FILE}"
    echo "Created user ${REGISTRY_USER}"
  fi
  kubectl create secret generic registry-auth \
    -n registry \
    --from-file="${HTPASSWD_FILE}" \
    --dry-run=client -o yaml | kubectl apply -f -
  # Use the checked-in deployment.yaml instead of a temp file
  kubectl apply -f "$(dirname "$0")/deployment.yaml" -n registry
  kubectl apply -f "$(dirname "$0")/service.yaml" -n registry
  kubectl apply -f "$(dirname "$0")/pvc.yaml" -n registry
  kubectl apply -f "$(dirname "$0")/serviceaccount.yaml" -n registry
  echo "Simple registry deployment complete!"
  echo "Registry will be accessible at: http://${REGISTRY_URL}"
  echo "Username: ${REGISTRY_USER}"
  echo "Password: ${REGISTRY_PW}"
  echo
  echo "You may need to run:"
  echo "  $0 configure-docker-simple  # To configure Docker for this simple registry"
}

function configure_docker_simple() {
  echo "Configuring Docker to use simple HTTP registry at ${REGISTRY_URL}..."
  mkdir -p "${DOCKER_CONFIG_DIR}" || true
  if [ ! -f "${DOCKER_CONFIG_FILE}" ]; then
    echo '{
  "insecure-registries": ["'${REGISTRY_URL}'", "registry.local:'${REGISTRY_PORT}'"]
}' >"${DOCKER_CONFIG_FILE}"
    echo "✅ Created new Docker daemon config file with insecure registry."
  else
    if grep -q "${REGISTRY_URL}" "${DOCKER_CONFIG_FILE}" && grep -q "registry.local:${REGISTRY_PORT}" "${DOCKER_CONFIG_FILE}"; then
      echo "✅ Registry already configured in Docker daemon."
    else
      cp "${DOCKER_CONFIG_FILE}" "${DOCKER_CONFIG_FILE}.bak"
      echo "📁 Backed up existing daemon.json to daemon.json.bak"
      TMP_FILE=$(mktemp)
      if ! grep -q "${REGISTRY_URL}" "${DOCKER_CONFIG_FILE}"; then
        jq '.["insecure-registries"] += ["'${REGISTRY_URL}'"]' "${DOCKER_CONFIG_FILE}" >"${TMP_FILE}" || {
          echo "⚠️ Failed to update daemon.json with jq, using simpler approach"
          sed 's/\("insecure-registries": \[\)/\1"'${REGISTRY_URL}'", /' "${DOCKER_CONFIG_FILE}"
        }
        mv "${TMP_FILE}" "${DOCKER_CONFIG_FILE}"
      fi
      if ! grep -q "registry.local:${REGISTRY_PORT}" "${DOCKER_CONFIG_FILE}"; then
        jq '.["insecure-registries"] += ["registry.local:'${REGISTRY_PORT}'"]' "${DOCKER_CONFIG_FILE}" >"${TMP_FILE}" || {
          echo "⚠️ Failed to update daemon.json with jq, using simpler approach"
          sed 's/\("insecure-registries": \[\)/\1"registry.local:'${REGISTRY_PORT}'", /' "${DOCKER_CONFIG_FILE}"
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
  docker run --rm --entrypoint htpasswd httpd:2 -Bbn "$1" "$2" >>"${HTPASSWD_FILE}"
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
  REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  if [ -n "${REGISTRY_POD}" ]; then
    echo "Restarting registry pod to apply changes..."
    kubectl delete pod "${REGISTRY_POD}" -n registry
  else
    echo "Registry pod not found. Is the registry deployed?"
  fi
}

init_directories

case "$1" in
install-simple)
  install_simple
  ;;
configure-docker-simple)
  configure_docker_simple
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
*)
  print_usage
  ;;
esac
