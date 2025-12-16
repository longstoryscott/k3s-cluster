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

## Networking Architecture (Detailed)

- **Edge / Reverse Proxy:** A public-facing reverse proxy (external to the cluster) terminates TLS for public hostnames (e.g. `auth.longstorymedia.com`) and forwards traffic to the NGINX Gateway LoadBalancer IP / nodePort on the LAN. Upstream connections from the reverse proxy to the gateway must use the correct scheme (`http://` for plain HTTP listeners) and preserve the `Host` header.
- **NGINX Gateway Fabric (in-cluster):** A LoadBalancer Service (`lsm-gateway-nginx`) exposes many listener ports on the cluster node IP (e.g. `192.168.0.71`) and maps each listener to an internal server block which proxies to a Kubernetes Service (port-based routing). Each `HTTPRoute` is bound to a `Gateway` listener via `sectionName` and `hostnames`.
- **Service Backends:** Services are normal `ClusterIP` services. Gateway configuration creates upstreams pointing to `<namespace>_<service>_<port>` (e.g. `auth_dex_5556`) and uses the cluster DNS name (`<service>.<namespace>.svc.cluster.local`) in controller diagnostics.
- **DNS and DDNS:** Public DNS (or DDNS) names must resolve to the router/public IP used by the reverse proxy. If your DDNS provider fails to resolve to your current public IP, the reverse proxy might attempt to route to an incorrect public address and return upstream 504s.

## Troubleshooting Network 504s (Step-by-step)

1. Reproduce the failing request and capture timestamps and client IP from upstream logs (external proxy). Note the exact request path and Host header.
2. Confirm `HTTPRoute` and `Gateway` are accepted and programmed:
  - `kubectl describe httproute dex-route -n nginx-gateway`
  - `kubectl describe gateway lsm-gateway -n nginx-gateway`
3. Confirm `Service` and `Endpoints` for the backend exist:
  - `kubectl get svc dex -n auth -o wide`
  - `kubectl get endpoints dex -n auth -o yaml`
4. From inside the `lsm-gateway-nginx` pod, test backend reachability and routing:
  - `kubectl exec -it <gateway-pod> -n nginx-gateway -- curl -vS http://dex.auth.svc.cluster.local:5556/.well-known/openid-configuration`
  - `kubectl exec -it <gateway-pod> -n nginx-gateway -- curl -vS -H "Host: auth.longstorymedia.com" http://127.0.0.1:9091/.well-known/openid-configuration`
  If these return 200, the in-cluster path and gateway routing are working.
5. From the reverse proxy (external) host, verify upstream connectivity and that the proxy is forwarding the Host header correctly:
  - `curl -vS -H "Host: auth.longstorymedia.com" http://<gateway-lan-ip>:9091/.well-known/openid-configuration`
  - `nc -vz <gateway-lan-ip> 9091`
  - If your reverse proxy attempts TLS to the upstream, use `openssl s_client -connect <gateway-lan-ip>:9091 -servername auth.longstorymedia.com` to validate TLS handshake.
6. Check for DNS/DDNS mismatches — if your public DDNS does not resolve to the current public IP, the reverse proxy or router may route to an incorrect address. Use `dig +short <your-ddns>` to verify.
7. If `curl` from the external host returns 200 but browsers see intermittent 504s, increase upstream timeouts on the external proxy and the gateway `ClientSettingsPolicy` timeout as needed.
8. If traffic never arrives at the gateway node, capture a short `tcpdump` on the gateway host while reproducing the client request to check whether SYN packets reach the host.

Common Fixes
- Ensure the external reverse proxy uses `http://` (not `https://`) when forwarding to plain HTTP gateway listeners.
- Preserve `Host` header in the proxy so the gateway can route by SNI/Host.
- If using a DDNS service, verify its resolution and TTL. If it intermittently resolves to a different IP (or fails), replace it with a stable public IP or a reliable DDNS provider until the issue is fixed.
- If you need TLS between reverse proxy and gateway, configure TLS on the gateway and update gateway listener to expect TLS.