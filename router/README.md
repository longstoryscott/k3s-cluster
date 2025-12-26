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

### 504 Errors / External Access Not Working

If services return 504 errors or are unreachable externally, follow this diagnostic flow:

**1. Verify internal connectivity first:**
```bash
# Direct to cluster (should return 302 for NextCloud)
curl -s -o /dev/null -w "%{http_code}" http://192.168.0.71:3000

# Via nodePort
kubectl get svc lsm-gateway-nginx -n nginx-gateway -o jsonpath='{.spec.ports[?(@.port==3000)].nodePort}'
curl -s -o /dev/null -w "%{http_code}" http://192.168.0.71:<NODEPORT>
```

**2. Check LoadBalancer is on the correct node:**
```bash
kubectl get svc lsm-gateway-nginx -n nginx-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
# Should be 192.168.0.71 (lsnode-0)
```

**3. If LoadBalancer IP is wrong, force svclb pod recreation:**
```bash
kubectl delete pods -n kube-system -l svccontroller.k3s.cattle.io/svcname=lsm-gateway-nginx
```

**4. If internal works but external doesn't:**
- Test from outside your network (mobile data) to rule out hairpin NAT
- Toggle the router port forwarding rule off and on
- Check router firewall settings
- Verify DDNS is resolving to correct public IP: `dig +short lsnet.tplinkdns.com`

**5. Nuclear option - reapply everything:**
```bash
make router
```

### Common Causes of Connectivity Issues

| Symptom | Likely Cause | Fix |
|---------|--------------|-----|
| 504 from reverse proxy | Port forward stale | Toggle port forward rule |
| Connection refused on public IP | svclb not bound | `make router` or delete svclb pods |
| LoadBalancer on wrong node | Pod scheduling | Already fixed via nodeSelector in NginxProxy |
| Works internally, fails externally | Hairpin NAT / ISP blocking | Test from external network |

### Architecture Notes

The Gateway uses k3s ServiceLB (klipper-lb) which creates svclb pods with hostPort bindings. 
The NginxProxy config pins these to `lsnode-0` (192.168.0.71) via nodeSelector to ensure 
router port forwarding rules remain valid.