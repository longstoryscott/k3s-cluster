# NGINX Gateway Fabric Manifests

This directory contains the modular Kubernetes manifests for NGINX Gateway Fabric v2.1.4, organized by resource type for better maintainability and clarity.

## File Structure

| File | Description | Resources |
|------|-------------|-----------|
| `01-namespace.yaml` | Namespace creation | Namespace |
| `02-serviceaccounts.yaml` | Service accounts | ServiceAccount × 2 |
| `03-rbac.yaml` | RBAC permissions | Role, ClusterRole, RoleBinding, ClusterRoleBinding |
| `04-service.yaml` | Control plane service | Service (ClusterIP) |
| `05-deployment.yaml` | Main controller | Deployment |
| `06-cert-job.yaml` | Certificate generator | Job |
| `07-gateway-config.yaml` | Gateway configuration | GatewayClass, NginxGateway, NginxProxy |
| `kustomization.yaml` | Kustomize configuration | - |

## Key Features

### Security
- **Non-root execution**: All containers run as non-root users
- **Read-only filesystem**: Containers use read-only root filesystems
- **Dropped capabilities**: All unnecessary Linux capabilities are dropped
- **Security contexts**: Proper security contexts with seccomp profiles

### Resource Management
- **Resource limits**: CPU and memory limits configured
- **Resource requests**: Baseline resource requirements defined
- **Graceful termination**: Proper termination grace periods

### Configuration
- **LoadBalancer service**: Automatically creates external IP via NginxProxy
- **External traffic policy**: Set to `Local` for better performance
- **Image pull policy**: `IfNotPresent` for efficiency
- **Logging level**: Set to `info` for operational visibility

## Usage

Deploy using kustomize:
```bash
kubectl apply -k .
```

Or use the parent install script:
```bash
../install.sh
```

## Resource Requirements

| Container | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-----------|-------------|----------------|-----------|--------------|
| nginx-gateway | 100m | 128Mi | 500m | 512Mi |
| cert-generator | - | - | - | - |

## Generated Resources

The NginxProxy configuration automatically creates:
- **Data plane deployment**: NGINX pods for traffic handling
- **LoadBalancer service**: External service `lsm-gateway-nginx`
- **TLS certificates**: Via the cert-generator job

## Dependencies

### Required CRDs
- Gateway API CRDs (applied by install script)
- NGINX Gateway Fabric CRDs (applied by install script)

### External Dependencies
- k3s servicelb (for LoadBalancer functionality)
- Docker registry access for images

## Troubleshooting

### Common Issues

**Job recreation needed:**
The cert-generator job is not idempotent and must be deleted before reapplying.

**Selector immutability:**
Deployment selectors cannot be changed. Full deletion/recreation required for selector changes.

**Image pull issues:**
Ensure cluster has internet access to pull from `ghcr.io/nginx/nginx-gateway-fabric`.

### Verification Commands

```bash
# Check all resources
kubectl get all -n nginx-gateway

# Check CRDs
kubectl get crd | grep gateway

# Check logs
kubectl logs -n nginx-gateway deployment/nginx-gateway

# Check LoadBalancer service
kubectl get svc -n nginx-gateway lsm-gateway-nginx
```