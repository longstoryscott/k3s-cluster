.DEFAULT_GOAL := all

export HELM_KUBECONTEXT=lsnet

.SILENT:

all:
	$(CURDIR)/install.sh $(CURDIR)

restart:
	kubectl delete --all all,configmap,secret,pvc,ingress,serviceaccount -n auth
	kubectl delete namespace auth
	$(CURDIR)/install.sh $(CURDIR)