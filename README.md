# INIT

1. create RSA keys between all rpi nodes
1. name master node and other nodes and update [k8s-up.sh](./scripts/k8s-up.sh)
1. `make`
1. Run private [registry](https://www.paulsblog.dev/how-to-install-a-private-docker-container-registry-in-kubernetes/)
1. run nginx [ingress](https://platform9.com/learn/v1.0/tutorials/nginix-controller-via-yaml) also [this](https://www.alibabacloud.com/blog/routing-applications-in-kubernetes-with-nginx-ingress_594719)

# OUTPUT

```sh
...installs...
customresourcedefinition.apiextensions.k8s.io/gatewayclasses.gateway.networking.k8s.io created
customresourcedefinition.apiextensions.k8s.io/gateways.gateway.networking.k8s.io created
customresourcedefinition.apiextensions.k8s.io/httproutes.gateway.networking.k8s.io created
customresourcedefinition.apiextensions.k8s.io/referencegrants.gateway.networking.k8s.io created
customresourcedefinition.apiextensions.k8s.io/nginxgateways.gateway.nginx.org created
namespace/nginx-gateway created
serviceaccount/nginx-gateway created
clusterrolebinding.rbac.authorization.k8s.io/nginx-gateway created
deployment.apps/nginx-gateway created
gatewayclass.gateway.networking.k8s.io/nginx created
nginxgateway.gateway.nginx.org/nginx-gateway-config created
enter username
lsm
enter password
.................+..+...
-----
namespace/registry created

## Networking Architecture (Overview)

- Edge reverse proxy: A public-facing reverse proxy (external to this cluster) terminates TLS for public hostnames and forwards traffic to the NGINX Gateway LoadBalancer IP and ports on the LAN. Upstream must use the correct scheme and preserve the Host header.
- In-cluster gateway: This repo uses NGINX Gateway Fabric with port-based listeners defined in `router/routes.yaml`. The LoadBalancer service `lsm-gateway-nginx` exposes listener ports on the node IP (e.g. `192.168.0.71`) and the gateway proxies to ClusterIP services like `dex` using backend upstream names such as `auth_dex_5556`.
- DNS / DDNS: Public DNS or DDNS names must resolve to the public IP used by the external reverse proxy. If a DDNS provider returns an incorrect IP, the external proxy could forward traffic to the wrong host and produce 504s.

## Troubleshooting Network 504s

1. Confirm Gateway and HTTPRoute are accepted and programmed:
	- `kubectl describe gateway lsm-gateway -n nginx-gateway`
	- `kubectl describe httproute dex-route -n nginx-gateway`
2. Check service and endpoints for the backend:
	- `kubectl get svc dex -n auth -o wide`
	- `kubectl get endpoints dex -n auth -o yaml`
3. From the gateway pod, test backend reachability:
	- `kubectl exec -it <gateway-pod> -n nginx-gateway -- curl -vS http://dex.auth.svc.cluster.local:5556/.well-known/openid-configuration`
	- `kubectl exec -it <gateway-pod> -n nginx-gateway -- curl -vS -H "Host: auth.longstorymedia.com" http://127.0.0.1:9091/.well-known/openid-configuration`
4. From the external reverse proxy host, verify the upstream connection and Host header forwarding:
	- `curl -vS -H "Host: auth.longstorymedia.com" http://<gateway-lan-ip>:9091/.well-known/openid-configuration`
	- `nc -vz <gateway-lan-ip> 9091`
	- `openssl s_client -connect <gateway-lan-ip>:9091 -servername auth.longstorymedia.com` (only if you expect TLS to the gateway)
5. Check router/NAT/port-forward mappings if the external IP does not reach the correct LAN host.

If you want help interpreting `curl`, `nc`, or `tcpdump` output, paste them here and I will analyze them.
service/registry created
deployment.apps/registry created
persistentvolume/registry-pv created
persistentvolumeclaim/registry-pvc created
secret/certs-secret created
secret/auth-secret created
gateway.gateway.networking.k8s.io/lsm-gateway created
referencegrant.gateway.networking.k8s.io/access-to-registry created
httproute.gateway.networking.k8s.io/registry-route created
```