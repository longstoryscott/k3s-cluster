#! /bin/bash

kubectl delete deployment coffee -n cafe
kubectl delete deployment tea -n cafe
kubectl delete service coffee -n cafe
kubectl delete service tea -n cafe
kubectl delete gateway gateway
kubectl delete referencegrant access-to-cafe-services -n cafe
kubectl delete httproute coffee
kubectl delete httproute tea
kubectl delete namespace cafe
