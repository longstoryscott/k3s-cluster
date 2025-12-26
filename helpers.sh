#!/usr/bin/env bash
set -e

SECRETS_DIR="${1}/.secrets"
mkdir -p "${SECRETS_DIR}"

# Load configuration from config.env if it exists
function load_config() {
    local config_file="$(dirname "$0")/config.env"
    if [[ -f "$config_file" ]]; then
        set -a
        source "$config_file"
        set +a
        echo "✓ Loaded configuration from config.env" >&2
    else
        echo "⚠️  Warning: config.env not found. Copy config.env.example and customize it." >&2
    fi
}

# Get a config value with fallback
function get_config() {
    local var_name=$1
    local default_value=$2
    echo "${!var_name:-$default_value}"
}

# Generate a random secret and store it in .secrets/
function gen_secret() {
    local name=$1
    local path="${SECRETS_DIR}/${name}"
    [[ -f "$path" ]] || openssl rand -hex 12 >"$path"

    echo "--from-file=${name}=${path}"
}

# Generate a password file if it doesn't exist
function gen_password_file() {
    local file_path=$1
    local length=${2:-16}
    
    if [[ ! -f "$file_path" ]]; then
        mkdir -p "$(dirname "$file_path")"
        openssl rand -hex "$length" > "$file_path"
        echo "✓ Generated password file: $file_path" >&2
    else
        echo "✓ Using existing password: $file_path" >&2
    fi
}
