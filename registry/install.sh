#!/usr/bin/env bash
set -e

# Always use the simple (HTTP, no TLS) registry setup for local development
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

"${SCRIPT_DIR}/registry-mgmt.sh" install-simple
"${SCRIPT_DIR}/registry-mgmt.sh" configure-docker-simple

echo "Simple local registry setup complete."
echo "Restart Docker Desktop if you haven't already."
