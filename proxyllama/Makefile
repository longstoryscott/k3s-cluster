export HELM_KUBECONTEXT=lsnet

.SILENT:

start:
	@export LOCAL=true && air

deploy:
	@chmod +x ./deploy.sh
	@echo "Deploying image..."
	@./deploy.sh $(CURDIR)