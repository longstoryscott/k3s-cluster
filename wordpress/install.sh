#! /bin/bash

helm upgrade --install wordpress oci://registry-1.docker.io/bitnamicharts/wordpress \
    --create-namespace -n wordpress \
    --set service.type=ClusterIP \
    --set ingress.enabled=true \
    --set ingress.hostname=knitfoxandfolk.local \
    --set hostAliases[0].ip=127.0.0.1 \
    --set hostAliases[0].hostnames[0]=knitfoxandfolk.local \
    --set wordpressEmail=foxandfolkknits@gmail.com \
    --set wordpressUsername=fnf \
    --set wordpressFirstName=Norah \
    --set wordpressLastName=Long \
    --set wordpressBlogName="Fox & Folk" \
    --set wordpressTablePrefix=fnf_ \
    --set readinessProbe.enabled=false \
    --values "${VALUES}"
