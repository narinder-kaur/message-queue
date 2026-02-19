#!/bin/bash
if echo $(kind get clusters) | grep "kind-cluster"; then
    echo "Cluster 'kind-cluster' already exists. Skipping cluster creation."
    exit 0
else
    echo "Cluster 'kind-cluster' does not exist. Proceeding with cluster creation."
    echo "Creating Kubernetes cluster with kind..."
    export HOSTPATH="$(PWD)/internal/data" && envsubst < ./kind/config.yaml | kind create cluster --name kind-cluster --config -
fi