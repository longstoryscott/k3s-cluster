#!/bin/bash

# Configuration substitution helper
# This script helps substitute domain names and IPs in configuration files

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/helpers.sh"
load_config

# Function to replace placeholders in a file
substitute_config() {
    local file=$1
    local temp_file="${file}.tmp"
    
    # Create a copy with substitutions
    sed \
        -e "s/{{MASTER_NODE_IP}}/${MASTER_NODE_IP}/g" \
        -e "s/{{REGISTRY_IP}}/${REGISTRY_IP}/g" \
        -e "s/{{REGISTRY_PORT}}/${REGISTRY_PORT}/g" \
        -e "s/{{AUTH_DOMAIN}}/${AUTH_DOMAIN}/g" \
        -e "s/{{DOMAIN_BASE}}/${DOMAIN_BASE}/g" \
        -e "s/{{DDNS_DOMAIN}}/${DDNS_DOMAIN}/g" \
        "$file" > "$temp_file"
    
    mv "$temp_file" "$file"
}

# Usage examples:
# substitute_config "router/routes.yaml"
# substitute_config "nc/configmap.yaml"
