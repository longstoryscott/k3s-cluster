# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

This is an authentication service deployed to a k3s cluster, providing OAuth2/OIDC authentication via Dex integrated with OpenLDAP for user storage. It includes a custom Go API (`usrmgr`) for user management operations.

## Architecture

### Components

1. **Dex** (`ghcr.io/dexidp/dex:v2.43.1`) - OAuth2/OIDC identity provider
   - Listens on port 5556
   - Uses LDAP connector for authentication against OpenLDAP
   - Configured with static clients including `lsm-client` and `public-client`
   - Custom web templates mounted from `/web/templates`

2. **OpenLDAP** (`osixia/openldap:1.5.0`) - LDAP directory server
   - Listens on port 389
   - Domain: `longstorymedia.com`
   - Organization: `Long Story Media`
   - Persistent storage via PVCs for data and config
   - Seeded with initial users/groups from LDIF files

3. **usrmgr** (custom Go binary) - User management API
   - Built with Fiber v3 framework
   - Listens on port 3333
   - Requires OAuth2 token authentication via Dex JWKS validation
   - Endpoints:
     - `GET /api/search` - Search LDAP users
     - `POST /api/user` - Add new user (requires `admins` group)
     - `PUT /api/password` - Change user password
     - `DELETE /api/user` - Delete user (requires `admins` group)

### Authentication Flow

1. Users authenticate via Dex UI or programmatic OAuth2 flow
2. Dex validates credentials against OpenLDAP
3. Dex issues JWT tokens signed with keys available at `/keys` endpoint
4. usrmgr validates JWT tokens using `keyfunc` library with JWKS URI
5. Authorization checks use groups claim from JWT (e.g., `admins` group for admin operations)

### Configuration

- Dex config uses `envsubst` for secret injection (`${DEX_CLIENT_SECRET}`, `${LDAP_ADMIN_PASSWORD}`)
- usrmgr uses environment variables (`USRMGR_*`) for configuration
- Secrets stored in Kubernetes secrets, LDIF in ConfigMaps

## Development Commands

### Build and Deploy

```bash
# Build and push image to private registry, then deploy
bash install.sh

# Build and push only (used by install.sh)
bash build-push.sh
```

### Local Development

```bash
# Navigate to API directory
cd api

# Build locally
CGO_ENABLED=0 GOOS=linux go build -o usrmgr .

# Run locally (requires environment variables set)
export USRMGR_LDAP_URL=...
export USRMGR_JWKS_URI=...
./usrmgr
```

### Kubernetes Operations

```bash
# View logs
kubectl logs -n auth -l app=dex
kubectl logs -n auth -l app=openldap
kubectl logs -n auth -l app=usrmgr

# Restart deployments
kubectl rollout restart deployment/dex -n auth
kubectl rollout restart deployment/openldap -n auth
kubectl rollout restart deployment/usrmgr -n auth

# Check rollout status
kubectl rollout status deployment/dex -n auth
```

## Key Files

- `deployment.yaml` - Kubernetes deployments for all three components
- `config.yaml` - Dex configuration template
- `api/main.go` - usrmgr entry point, route registration
- `api/auth/validator.go` - JWT validation middleware
- `api/handlers/*.go` - API endpoint handlers
- `api/config/*.go` - Configuration structs and helpers
- `web/templates/` - Custom Dex HTML templates
- `ldap-users.ldif`, `ldap-groups.ldif` - Initial LDAP data

## Important Notes

- The Go module is named `usrmgr` (see `api/go.mod`)
- All API routes require Bearer token authentication except OPTIONS
- Admin operations check for `admins` group in JWT claims
- LDAP uses `inetOrgPerson` objectClass for users
- Private registry at `192.168.0.71:31500` requires credentials from `../registry/.secrets/`

## External Access Architecture

Traffic flow for external clients:
1. Client → `auth.longstorymedia.com` (public DNS)
2. → Proxy server at `159.89.45.33:443` (nginx reverse proxy with TLS termination)
3. → Home router public IP `73.94.53.57` (ports 9091/3333 forwarded)
4. → k3s master node `192.168.0.71` (LAN IP)
5. → NGINX Gateway Fabric routes to backend services

The proxy server's nginx config at `/etc/nginx/sites-available/auth` proxies:
- `/` → `http://73.94.53.57:9091` (Dex on port 9091)
- `/api` → `http://73.94.53.57:3333` (usrmgr on port 3333)