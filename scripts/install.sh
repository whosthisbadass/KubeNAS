#!/usr/bin/env bash
set -euo pipefail

kubectl apply -f deploy/crds/
kubectl apply -f deploy/operator.yaml
kubectl apply -f deploy/examples/node-agent-daemonset.yaml

echo "KubeNAS manifests applied."
