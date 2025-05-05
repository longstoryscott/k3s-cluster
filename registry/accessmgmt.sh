#!/usr/bin/env bash
set -e

REGISTRY_HOME="${HOME}/.registry"
CERTS_DIR="${REGISTRY_HOME}/certs"
AUTH_DIR="${REGISTRY_HOME}/auth"
HTPASSWD_FILE="${AUTH_DIR}/htpasswd"
REGISTRY_HOSTNAME="registry.local"

function print_usage() {
    cat <<EOF
Registry Access Management Utility

Usage:
  ./accessmgmt.sh [command]

Available Commands:
  add-user    <username> <password>   - Add a new user to the registry
  del-user    <username>              - Delete a user from the registry
  change-pw   <username> <password>   - Change a user's password
  list-users                          - List all users in the registry
  renew-cert                          - Renew the TLS certificate

Examples:
  ./accessmgmt.sh add-user developer password123
  ./accessmgmt.sh del-user developer
  ./accessmgmt.sh renew-cert

EOF
}

function init_directories() {
    mkdir -p "${REGISTRY_HOME}" "${CERTS_DIR}" "${AUTH_DIR}" || true
    touch "${HTPASSWD_FILE}" || true
    chmod 600 "${HTPASSWD_FILE}" || true
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

    # Restart the registry pod to apply changes
    REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}')
    if [ -n "${REGISTRY_POD}" ]; then
        kubectl delete pod "${REGISTRY_POD}" -n registry
        echo "Registry pod restarting to apply changes..."
    fi
}

function renew_certificate() {
    echo "Generating new self-signed certificates..."

    # Create a backup of the old certificates
    if [ -f "${CERTS_DIR}/tls.key" ] && [ -f "${CERTS_DIR}/tls.crt" ]; then
        BACKUP_DIR="${CERTS_DIR}/backup-$(date +%Y%m%d-%H%M%S)"
        mkdir -p "${BACKUP_DIR}"
        cp "${CERTS_DIR}/tls.key" "${BACKUP_DIR}/tls.key"
        cp "${CERTS_DIR}/tls.crt" "${BACKUP_DIR}/tls.crt"
        echo "Backed up old certificates to ${BACKUP_DIR}"
    fi

    # Generate new certificates
    openssl req -x509 \
        -newkey rsa:4096 \
        -days 365 -nodes -sha256 \
        -keyout "${CERTS_DIR}/tls.key" \
        -out "${CERTS_DIR}/tls.crt" \
        -subj "/CN=${REGISTRY_HOSTNAME}" \
        -addext "subjectAltName = DNS:${REGISTRY_HOSTNAME},DNS:registry,IP:127.0.0.1"

    # Update the Kubernetes secret
    kubectl create secret tls registry-certs-secret \
        -n registry \
        --cert="${CERTS_DIR}/tls.crt" \
        --key="${CERTS_DIR}/tls.key" \
        --dry-run=client -o yaml | kubectl apply -f -

    # Restart the registry pod to apply changes
    REGISTRY_POD=$(kubectl get pods -n registry -l app=registry -o jsonpath='{.items[0].metadata.name}')
    if [ -n "${REGISTRY_POD}" ]; then
        kubectl delete pod "${REGISTRY_POD}" -n registry
        echo "Registry pod restarting to apply changes..."
    fi

    echo "TLS certificate renewed successfully."
    echo "Remember to update client certificates:"
    echo "sudo mkdir -p /etc/docker/certs.d/${REGISTRY_HOSTNAME}:5000/"
    echo "sudo cp ${CERTS_DIR}/tls.crt /etc/docker/certs.d/${REGISTRY_HOSTNAME}:5000/ca.crt"
}

# Main execution
init_directories

case "$1" in
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
renew-cert)
    renew_certificate
    ;;
*)
    print_usage
    ;;
esac
