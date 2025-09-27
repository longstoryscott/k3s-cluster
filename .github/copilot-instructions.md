# Copilot Instructions for k3s-cluster

## Architecture Overview
This is a multi-node Raspberry Pi k3s cluster with a microservices architecture. Services are deployed using kubectl manifests and managed through the central `Makefile` orchestration system.

**Key Components:**
- **NGINX Gateway Fabric**: Gateway API implementation for ingress routing (port-based routing)
- **Private Docker Registry**: Self-hosted at `registry.local:31500` with UI at `:31501`
- **Service Stack**: Ollama (AI), Grafana/Prometheus (monitoring), NextCloud, PostgreSQL/MySQL databases
- **Node Architecture**: Master node (`lsnode-0`) + 3 worker nodes (`lsnode-1` through `lsnode-3`)

## Essential Workflows

### Cluster Management
```bash
# Full cluster setup
make install

# Individual services
make registry    # Deploy private registry
make ollama      # Deploy Ollama AI service  
make monitoring  # Deploy Grafana/Prometheus stack
make nc          # Deploy NextCloud

# Registry operations
./registry/registry-mgmt.sh install-simple
./registry/registry-mgmt.sh add-user <username> <password>
```

### Service Deployment Pattern
Each service follows this structure:
```
service/
├── install.sh          # Main deployment script
├── deployment.yaml     # K8s deployment
├── service.yaml        # K8s service
├── pvc.yaml           # Persistent volume claim
└── referencegrant.yaml # Gateway API cross-namespace refs
```

All `install.sh` scripts:
1. Create namespace with `kubectl create namespace <service> || true`
2. Apply manifests in order: PVC → ConfigMaps → Deployment → Service → ReferenceGrant
3. Use `--wait=true` for sequential deployment validation

## Project-Specific Conventions

### Gateway API Routing
Services are exposed through NGINX Gateway with **port-based routing** (not hostname):
- Grafana: `:3001`
- Prometheus: `:9090` 
- Registry: `:5000`
- NextCloud: `:3000`

Routes configured in `router/routes.yaml` with this pattern:
```yaml
listeners:
  - name: service-name
    port: XXXX
    protocol: HTTP
```

### Secret Management
Secrets generated via `helpers.sh`:
```bash
# Creates .secrets/ directory with random hex values
gen_secret "secretname"  # Outputs kubectl --from-file flag
```

### Registry Integration
All custom images use the private registry `registry.local:31500`. Before pushing:
```bash
make docker-login  # Uses credentials from registry/.secrets/
```

### Database Services
Database deployments use StatefulSets with:
- `dbs.yaml` for database initialization
- `serviceaccount.yaml` for RBAC
- Persistent storage for data retention

### GPU Workloads
Ollama and AI services use:
```yaml
runtimeClassName: nvidia  # Enables GPU access
```

## Integration Patterns

### Cross-Service Communication
Services communicate via cluster DNS: `<service>.<namespace>.svc.cluster.local:port`

### Monitoring Integration  
All services automatically discovered by Prometheus via ServiceMonitor annotations. Logs collected by Promtail → Loki → Grafana.

### Authentication Flow
Dex (auth service) provides OIDC for NextCloud and other services requiring authentication.

## Development Tips

- Always run `make ex` first to ensure scripts are executable
- Use `kubectl apply -f file.yaml --wait=true` for reliable deployments
- Check Gateway routes with: `kubectl get httproutes -A`
- Monitor deployments: `kubectl rollout status deployment/<name> -n <namespace>`
- Registry debugging: `make registry-ls-<image>` to list tags

When adding new services, follow the established pattern: create service directory with standard files, add to Makefile, and configure Gateway route if external access needed.