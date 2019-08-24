#!/bin/sh

set -x
set -o errexit
set -o nounset

sudo sysctl -w vm.max_map_count=262144

curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.10.0/bin/linux/amd64/kubectl
chmod +x kubectl && sudo mv kubectl /usr/local/bin/
curl -Lo minikube https://storage.googleapis.com/minikube/releases/v0.25.2/minikube-linux-amd64
chmod +x minikube && sudo mv minikube /usr/local/bin/

sudo minikube start --vm-driver=none --kubernetes-version=v1.10.0
curl -OJL https://github.com/operator-framework/operator-sdk/releases/download/v0.7.0/operator-sdk-v0.7.0-x86_64-linux-gnu
chmod +x operator-sdk-v0.7.0-x86_64-linux-gnu && sudo mv operator-sdk-v0.7.0-x86_64-linux-gnu /usr/local/bin/operator-sdk

minikube update-context
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do
  sleep 1; done
