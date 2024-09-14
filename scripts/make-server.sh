#!/bin/bash

IP=$(./set-static-ip.sh)

./k8s-up.sh "$IP"
