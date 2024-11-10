#! /bin/bash

helm upgrade --install wordpress oci://registry-1.docker.io/bitnamicharts/wordpress \
    --create-namespace -n wordpress \
    --values "${VALUES}"
