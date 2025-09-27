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
clusterrole.rbac.authorization.k8s.io/nginx-gateway created
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