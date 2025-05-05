SCRIPTS = $(CURDIR)/scripts
REGISTRY = $(CURDIR)/registry
ROUTER = $(CURDIR)/router
REDIS = $(CURDIR)/redis
NEXTCLOUD = $(CURDIR)/nc
POSTGRESQL = $(CURDIR)/psql
MYSQL = $(CURDIR)/mysql
FNF = $(CURDIR)/fnf
OLLAMA = $(CURDIR)/ollama
AUTH = $(CURDIR)/auth
PROXYLLAMA = $(CURDIR)/proxyllama
MONITORING = $(CURDIR)/monitoring

export HELM_KUBECONTEXT=lsnet
export NODES=(lsnode-0 lsnode-1 lsnode-2 lsnode-3)
export MASTER_NODE=lsnode-0
export WORKER_NODES=(lsnode-1 lsnode-2 lsnode-3)

.SILENT:

install: init registry ex auth nc wp ollama router registry

init: ex
	cd $(SCRIPTS) && bash make-server.sh && cd ..

update-workers: ex
	cd $(SCRIPTS) && bash update-workers.sh && cd ..

ex:
	for f in $(SCRIPTS)/*.sh $(ROUTER)/*.sh $(REGISTRY)/*.sh $(NEXTCLOUD)/*.sh $(FNF)/*.sh $(MYSQL)/*.sh $(AUTH)/*sh $(PROXYLLAMA)/*sh $(MONITORING)/*.sh; do chmod +x "$$f"; done;

auth: ex
	$(AUTH)/install.sh $(AUTH)

registry:
	$(REGISTRY)/registry-mgmt.sh install

registry-%:
	$(REGISTRY)/registry-mgmt.sh $*

registry-help:
	$(REGISTRY)/registry-mgmt.sh

registry-user:
	$(REGISTRY)/registry-mgmt.sh manage-users add

registry-cert:
	$(REGISTRY)/registry-mgmt.sh manage-certs
	
configure-docker:
	$(REGISTRY)/registry-mgmt.sh configure-docker

docker-login:
	REGISTRY_USER=$$(cat $(CURDIR)/registry/.secrets/registryuser) && REGISTRY_PW=$$(cat $(CURDIR)/registry/.secrets/registrypw) && echo $$REGISTRY_PW | docker login https://registry.local -u $$REGISTRY_USER --password-stdin

router: ex
	$(ROUTER)/install.sh $(ROUTER)

psql:
	$(POSTGRESQL)/install.sh $(POSTGRESQL)

mysql:
	$(MYSQL)/install.sh $(MYSQL)

nc: ex
	$(NEXTCLOUD)/install.sh $(NEXTCLOUD)

fnf: ex
	$(FNF)/install.sh $(FNF)

ollama:
	$(OLLAMA)/install.sh $(OLLAMA)

proxyllama:
	$(PROXYLLAMA)/install.sh $(PROXYLLAMA)

monitoring: ex
	$(MONITORING)/install.sh $(MONITORING)

destroy:
	$(SCRIPTS)/k8s-down.sh

sync:
	rsync -avzru --delete --exclude=".git" --exclude="node_modules" -e 'ssh -p 2222' . lsm@lsnet.tplinkdns.com:~/k8s/

apply: sync
	ssh -p 2222 lsm@lsnet.tplinkdns.com "cd k8s && make"

.PHONY: router ex ollama psql mysql proxyllama nc monitoring fnf registry registry-user registry-renew-cert configure-docker registry-trust-cert