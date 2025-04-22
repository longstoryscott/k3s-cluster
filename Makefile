SCRIPTS = $(CURDIR)/scripts
INGRESS = $(CURDIR)/ingress
REGISTRY = $(CURDIR)/registry
ROUTER = $(CURDIR)/router
REDIS = $(CURDIR)/redis
NEXTCLOUD = $(CURDIR)/nextcloud
POSTGRESQL = $(CURDIR)/psql
WORDPRESS = $(CURDIR)/wordpress
OLLAMA = $(CURDIR)/ollama
AUTH = $(CURDIR)/auth
PROXYLLAMA = $(CURDIR)/proxyllama

.DEFAULT_GOAL := all

export HELM_KUBECONTEXT=lsnet

.SILENT:

all: ex auth nc wp ollama router

install: init gateway install-registry all

init: ex
	cd $(SCRIPTS) && bash make-server.sh && cd ..

update-workers: ex
	cd $(SCRIPTS) && bash update-workers.sh && cd ..

auth: ex
	$(AUTH)/install.sh $(AUTH)

restart-auth:
	kubectl delete --all all,configmap,secret,pvc,ingress,serviceaccount -n auth
	kubectl delete namespace auth
	$(AUTH)/install.sh $(AUTH)
	make router

ex:
	for f in $(SCRIPTS)/*.sh $(INGRESS)/*.sh $(REGISTRY)/*.sh $(NEXTCLOUD)/*.sh $(WORDPRESS)/*.sh $(AUTH)/*sh; do chmod +x "$$f"; done;

install-registry:
	$(REGISTRY)/install.sh $(REGISTRY) "patch"

update-registry:
	$(REGISTRY)/install.sh $(REGISTRY) 

gateway:
	kubectl kustomize "https://github.com/nginxinc/nginx-gateway-fabric/config/crd/gateway-api/standard?ref=v1.4.0" | kubectl apply -f -
	kubectl kustomize "https://github.com/nginxinc/nginx-gateway-fabric/config/crd/gateway-api/experimental?ref=v1.4.0" | kubectl apply -f -
	# kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml
	# kubectl apply -f https://github.com/nginxinc/nginx-gateway-fabric/releases/download/v1.2.0/crds.yaml

router:
	helm upgrade --install nginx-gateway oci://ghcr.io/nginxinc/charts/nginx-gateway-fabric --create-namespace -n nginx-gateway --values $(ROUTER)/values.yaml
	kubectl wait --timeout=5m -n nginx-gateway deployment/nginx-gateway-nginx-gateway-fabric --for=condition=Available
	kubectl apply -f $(ROUTER)/routes.yaml

psql:
	$(POSTGRESQL)/install.sh $(POSTGRESQL)

nc: ex
	$(NEXTCLOUD)/install.sh $(NEXTCLOUD)/values.yaml

nc-template: ex
	$(NEXTCLOUD)/install.sh $(NEXTCLOUD)/values.yaml template

delete-nc:
	$(NEXTCLOUD)/delete.sh

wp: ex
	$(WORDPRESS)/install.sh $(WORDPRESS)/values.yaml

ollama:
	$(OLLAMA)/install.sh $(OLLAMA)

proxyllama:
	$(PROXYLLAMA)/deploy.sh $(PROXYLLAMA)

destroy:
	$(SCRIPTS)/k8s-down.sh

sync:
	rsync -avzru --delete --exclude=".git" --exclude="node_modules" -e 'ssh -p 2222' . lsm@lsnet.tplinkdns.com:~/k8s/

apply: sync
	ssh -p 2222 lsm@lsnet.tplinkdns.com "cd k8s && make"

.PHONY: router ex ollama psql proxyllama