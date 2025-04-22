#!/usr/bin/env bash
set -e

SECRETS_DIR="${1}/.secrets"
mkdir -p "${SECRETS_DIR}"

function gen_secret() {
    local name=$1
    local path="${SECRETS_DIR}/${name}"
    [[ -f "$path" ]] || openssl rand -hex 32 >"$path"

    echo "--from-file=${name}=${path}"
}
