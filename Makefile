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

.DEFAULT_GOAL := all

export HELM_KUBECONTEXT=lsnet

.SILENT:

all: ex auth nc wp ollama router

install: init install-registry all

init: ex
	cd $(SCRIPTS) && bash make-server.sh && cd ..

update-workers: ex
	cd $(SCRIPTS) && bash update-workers.sh && cd ..

ex:
	for f in $(SCRIPTS)/*.sh $(ROUTER)/*.sh $(REGISTRY)/*.sh $(NEXTCLOUD)/*.sh $(FNF)/*.sh $(MYSQL)/*.sh $(AUTH)/*sh $(MONITORING)/*.sh; do chmod +x "$$f"; done;

auth: ex
	$(AUTH)/install.sh $(AUTH)

install-registry:
	$(REGISTRY)/install.sh $(REGISTRY) "patch"

update-registry:
	$(REGISTRY)/install.sh $(REGISTRY) 

router: ex
	$(ROUTER)/install.sh $(ROUTER)

psql:
	$(POSTGRESQL)/install.sh $(POSTGRESQL)

mysql:
	$(MYSQL)/install.sh $(MYSQL)

nc: ex
	$(NEXTCLOUD)/install.sh $(NEXTCLOUD)

delete-nc:
	$(NEXTCLOUD)/delete.sh

fnf: ex
	$(FNF)/install.sh $(FNF)

ollama:
	$(OLLAMA)/install.sh $(OLLAMA)

proxyllama:
	$(PROXYLLAMA)/deploy.sh $(PROXYLLAMA)

monitoring: ex
	$(MONITORING)/install.sh $(MONITORING)

destroy:
	$(SCRIPTS)/k8s-down.sh

sync:
	rsync -avzru --delete --exclude=".git" --exclude="node_modules" -e 'ssh -p 2222' . lsm@lsnet.tplinkdns.com:~/k8s/

apply: sync
	ssh -p 2222 lsm@lsnet.tplinkdns.com "cd k8s && make"

.PHONY: router ex ollama psql mysql proxyllama nc monitoring fnf