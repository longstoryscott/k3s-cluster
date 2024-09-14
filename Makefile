SCRIPTS = $(CURDIR)/scripts
INGRESS = $(CURDIR)/ingress
REGISTRY = $(CURDIR)/registry
ROUTER = $(CURDIR)/router
REDIS = $(CURDIR)/redis
NEXTCLOUD = $(CURDIR)/nextcloud
POSTGRESQL = $(CURDIR)/postgresql

.DEFAULT_GOAL := all

.SILENT:

all: ex nc router

install: init gateway install-registry nc router

init: ex
	cd $(SCRIPTS) && bash make-server.sh && cd ..

update-workers: ex
	cd $(SCRIPTS) && bash update-workers.sh && cd ..

ex:
	for f in $(SCRIPTS)/*.sh $(INGRESS)/*.sh $(REGISTRY)/*.sh $(NEXTCLOUD)/*.sh; do chmod +x "$$f"; done;

install-registry:
	$(REGISTRY)/install.sh $(REGISTRY) "patch"

update-registry:
	$(REGISTRY)/install.sh $(REGISTRY) 

gateway:
	kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.0.0/standard-install.yaml
	kubectl apply -f https://github.com/nginxinc/nginx-gateway-fabric/releases/download/v1.2.0/crds.yaml

router:
	helm template nginx-gateway oci://ghcr.io/nginxinc/charts/nginx-gateway-fabric --create-namespace -n nginx-gateway --values $(ROUTER)/values.yaml > $(ROUTER)/patch/manifest.yaml
	# helm upgrade --install nginx-gateway oci://ghcr.io/nginxinc/charts/nginx-gateway-fabric --create-namespace -n nginx-gateway --values $(ROUTER)/values.yaml
	# kubectl wait --timeout=5m -n nginx-gateway deployment/nginx-gateway-nginx-gateway-fabric --for=condition=Available
	kubectl apply -k $(ROUTER)/patch
	kubectl apply -f $(ROUTER)/routes.yaml

connect-postgres:
	kubectl run postgresql-client --rm --tty -i --restart='Never' --namespace postgresql --image docker.io/bitnami/postgresql:16.2.0-debian-12-r10 --env="PGPASSWORD=$(shell kubectl get secret --namespace nextcloud nextcloud-postgresql -o jsonpath='{.data.password}' | base64 -d)" --command -- psql --host postgresql -U lsm -d postgres -p 5432

nc: ex
	$(NEXTCLOUD)/install.sh $(NEXTCLOUD)/values.yaml

nc-template: ex
	$(NEXTCLOUD)/install.sh $(NEXTCLOUD)/values.yaml template

delete-nc:
	$(NEXTCLOUD)/delete.sh

destroy:
	$(SCRIPTS)/k8s-down.sh

sync:
	rsync -avzru --delete --exclude=".git" -e 'ssh -p 2222' . lsm@lsnet.tplinkdns.com:~/k8s/

apply: sync
	ssh -p 2222 lsm@lsnet.tplinkdns.com "cd k8s && make"

.PHONY: router ex