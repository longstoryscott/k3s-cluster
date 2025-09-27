#!/usr/bin/env bash

echo "=== Checking and fixing nginx proxy configuration ==="

UI_POD=$(kubectl get pods -n registry -l app=registry-ui -o jsonpath='{.items[0].metadata.name}')

if [ -n "${UI_POD}" ]; then
    echo "Checking full nginx configuration in UI pod:"
    kubectl exec "${UI_POD}" -n registry -- cat /etc/nginx/conf.d/default.conf
    echo
    echo "=== Checking if proxy configuration exists ==="
    kubectl exec "${UI_POD}" -n registry -- grep -A 10 -B 5 "location /v2" /etc/nginx/conf.d/default.conf || echo "No /v2 proxy location found!"
    echo
    echo "=== Checking environment variables used by nginx ==="
    kubectl exec "${UI_POD}" -n registry -- env | grep NGINX
else
    echo "UI pod not found"
fi

echo
echo "The issue appears to be that the nginx proxy configuration for /v2/ is missing."
echo "Let's restart the UI pod to ensure the nginx config is properly generated:"
kubectl delete pod -n registry -l app=registry-ui

echo "Waiting for new pod to start..."
kubectl wait --for=condition=ready pod -l app=registry-ui -n registry --timeout=60s

echo "New pod started. Let's check if the configuration is now correct:"
sleep 5

NEW_UI_POD=$(kubectl get pods -n registry -l app=registry-ui -o jsonpath='{.items[0].metadata.name}')
if [ -n "${NEW_UI_POD}" ]; then
    echo "Checking nginx config in new pod:"
    kubectl exec "${NEW_UI_POD}" -n registry -- grep -A 15 "location /v2" /etc/nginx/conf.d/default.conf || echo "Still no /v2 proxy location!"
fi