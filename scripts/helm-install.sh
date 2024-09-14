#! /bin/bash

if [ -z "$4" ]; then
    helm upgrade --install "${1}" "${2}" \
        --values "${3}"
else
    helm upgrade --install "${1}" "${2}" \
        --create-namespace -n "${4}" \
        --values "${3}"
fi
