# Router Directory

This directory contains the NGINX Gateway Fabric configuration for the k3s cluster.

## Files

- **`install.sh`** - Idempotent installation script for NGINX Gateway Fabric
- **`manifests/`** - Modular Kubernetes manifests organized by resource type
- **`routes.yaml`** - Gateway and HTTPRoute configurations for all services

## Usage

Deploy or update the NGINX Gateway Fabric:

```bash
make router
```

Or run directly:

```bash
./router/install.sh
```

## Features

- **LoadBalancer Service**: Automatically creates external IP on `192.168.0.71`
- **Port-based Routing**: Services accessible on different ports
- **Idempotent**: Can be run multiple times safely
- **CORS Support**: Automatic CORS headers for API endpoints
- **Timeout Policies**: Configured for long-running requests

## Service Ports

| Service      | Port | Description                    |
|--------------|------|--------------------------------|
| NextCloud    | 3000 | Main NextCloud instance        |
| Auth (Dex)   | 9091 | OpenID Connect authentication  |
| User Manager | 3333 | User management interface      |
| Grafana      | 3001 | Monitoring dashboard           |
| Prometheus   | 9090 | Metrics collection             |
| Registry     | 5000 | Docker registry                |
| Registry UI  | 8085 | Docker registry web interface  |
| PgAdmin      | 8084 | PostgreSQL admin interface     |
| SearXNG      | 8086 | Search engine                  |

## External Access

Services are accessible externally through:

- Router port forwarding: `192.168.0.71:PORT`
- Reverse proxy: `https://[subdomain].longstorymedia.com`

## Troubleshooting

Check Gateway status:

```bash
kubectl get gateway lsm-gateway -n nginx-gateway -o wide
```

Check HTTPRoute status:

```bash
kubectl get httproute -n nginx-gateway
```

Check LoadBalancer service:

```bash
kubectl get svc lsm-gateway-nginx -n nginx-gateway
```