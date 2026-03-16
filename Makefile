SCRIPTS = $(CURDIR)/scripts
REGISTRY = $(CURDIR)/registry
ROUTER = $(CURDIR)/router
REDIS = $(CURDIR)/redis
RABBITMQ = $(CURDIR)/rabbitmq
NEXTCLOUD = $(CURDIR)/nc
POSTGRESQL = $(CURDIR)/psql
MYSQL = $(CURDIR)/mysql
FNF = $(CURDIR)/fnf
OLLAMA = $(CURDIR)/ollama
AUTH = $(CURDIR)/auth
MONITORING = $(CURDIR)/monitoring
NVIDIA = $(CURDIR)/nvidia
AILAB = $(CURDIR)/ls-ai-ui
SEARXNG = $(CURDIR)/searxng
# GAMING_DESKTOP = $(CURDIR)/gaming-desktop
STEAM = $(CURDIR)/steam

export HELM_KUBECONTEXT=default
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
	for f in $(SCRIPTS)/*.sh $(ROUTER)/*.sh $(REGISTRY)/*.sh $(NEXTCLOUD)/*.sh $(FNF)/*.sh $(MYSQL)/*.sh $(AUTH)/*sh $(MONITORING)/*.sh $(RABBITMQ)/*.sh $(SEARXNG)/*.sh $(STEAM)/*.sh; do chmod +x "$$f"; done;

auth: ex
	$(AUTH)/install.sh $(AUTH)

registry:
	$(REGISTRY)/registry-mgmt.sh install-simple

registry-help:
	$(REGISTRY)/registry-mgmt.sh

registry-user:
	$(REGISTRY)/registry-mgmt.sh manage-users add

registry-ls-%:
	curl -s -u $$(cat $(CURDIR)/registry/.secrets/registryuser):$$(cat $(CURDIR)/registry/.secrets/registrypw) http://192.168.0.71:31500:31500/v2/$*/tags/list

configure-docker:
	$(REGISTRY)/registry-mgmt.sh configure-docker

docker-login:
	REGISTRY_USER=$$(cat $(CURDIR)/registry/.secrets/registryuser) && REGISTRY_PW=$$(cat $(CURDIR)/registry/.secrets/registrypw) && echo $$REGISTRY_PW | docker login 192.168.0.71:31500:31500 -u $$REGISTRY_USER --password-stdin

router: ex
	$(ROUTER)/install.sh

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

redis:
	$(REDIS)/install.sh $(REDIS)

rabbitmq:
	$(RABBITMQ)/install.sh $(RABBITMQ)

proxyllama:
	$(PROXYLLAMA)/install.sh $(PROXYLLAMA)

proxyllama-push:
	docker build -t registry.local:31500/proxyllama:latest $(PROXYLLAMA)
	docker tag registry.local:31500/proxyllama:latest registry.local:31500/proxyllama:$(shell date +%Y%m%d)
	docker push registry.local:31500/proxyllama:latest
	docker push registry.local:31500/proxyllama:$(shell date +%Y%m%d)

ai-lab:
	$(AILAB)/deploy.sh

monitoring: ex
	$(MONITORING)/install.sh $(MONITORING)

nvidia:
	$(NVIDIA)/install.sh

destroy:
	$(SCRIPTS)/k8s-down.sh

sync:
	rsync -avzru --delete --exclude=".git" --exclude="node_modules" -e 'ssh -p 22' . lsm@lsnode-0.local:~/k8s/

apply: sync
	ssh -p 2222 lsm@lsnet.tplinkdns.com "cd k8s && make"

registry-simple: ex
	$(REGISTRY)/registry-mgmt.sh install-simple
	$(REGISTRY)/registry-mgmt.sh configure-docker-simple

searxng:
	$(SEARXNG)/install.sh $(SEARXNG)

gaming-desktop: ex
	$(GAMING_DESKTOP)/install.sh $(GAMING_DESKTOP)

gaming-desktop-build:
	$(GAMING_DESKTOP)/build-push.sh

gaming-desktop-remote:
	$(GAMING_DESKTOP)/setup-remote-build.sh

steam: ex
	$(STEAM)/deploy.sh

.PHONY: router ex ollama psql mysql proxyllama nc monitoring fnf registry redis rabbitmq auth nvidia searxng gaming-desktop gaming-desktop-build gaming-desktop-remote steam