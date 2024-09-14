#! /bin/bash

kubectl delete svc -n nextcloud nextcloud-postgresql-hl
kubectl delete svc -n nextcloud nextcloud-postgresql
kubectl delete svc -n nextcloud nextcloud-redis-master
kubectl delete svc -n nextcloud nextcloud-redis-headless
kubectl delete svc -n nextcloud nextcloud-redis-replicas
kubectl delete svc nextcloud -n nextcloud
kubectl delete deployment nextcloud -n nextcloud
kubectl delete pod -n nextcloud nextcloud-redis-replicas-0
kubectl delete pod -n nextcloud nextcloud-redis-master-0
kubectl delete pod -n nextcloud nextcloud-postgresql-0
