#!/bin/bash

REGISTRY_DRIVE="$HOME/.registry"
CERTS_DRIVE="$REGISTRY_DRIVE/certs"
AUTH_DRIVE="$REGISTRY_DRIVE/auth"
PW_FILE="$REGISTRY_DRIVE/auth/htpasswd"

function registry-init-secrets() {
    mkdir -p "$REGISTRY_DRIVE" >/dev/null 2>&1
    mkdir -p "$CERTS_DRIVE" >/dev/null 2>&1
    mkdir -p "$AUTH_DRIVE" >/dev/null 2>&1
    touch "$PW_FILE" >/dev/null 2>&1
    chmod 600 "$PW_FILE" >/dev/null 2>&1
}

function generate-cert() {
    openssl req -x509 \
        -newkey rsa:4096 \
        -days 365 -nodes -sha256 \
        -keyout $CERTS_DRIVE/tls.key \
        -out $CERTS_DRIVE/tls.crt \
        -subj "/CN=registry" \
        -addext "subjectAltName = DNS:registry"
}

function k8s-secrets() {
    kubectl create secret tls certs-secret \
        -n registry \
        --cert=$CERTS_DRIVE/tls.crt \
        --key=$CERTS_DRIVE/tls.key
    kubectl create secret generic auth-secret \
        -n registry \
        --from-file=$PW_FILE
}

function registry-add-user() {
    docker run --rm \
        --entrypoint htpasswd \
        httpd:2 -Bbn "${1}" "${2}" >>"$PW_FILE"
}

function registry-del-user() {
    local line
    line=$(grep "${1}" "$PW_FILE")
    sed -i "s|${line}||g" "$PW_FILE"
    sed -i '/^$/d' "$PW_FILE"
}

function registry-chpw() {
    registry-del-user "${1}"
    registry-add-user "${1}" "${2}"
}
