#!/usr/bin/env bash
# Spins up a local kind cluster running pulse-api + mongo + redis, with an
# HPA watching pulse-api's CPU. Run from the repo root inside the Codespace:
#
#   bash k8s/bootstrap.sh
#
set -e

CLUSTER_NAME=pulse

echo "==> Checking for kind/kubectl..."
if ! command -v kind >/dev/null 2>&1; then
  echo "    Installing kind..."
  curl -Lo /tmp/kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-amd64
  chmod +x /tmp/kind
  sudo mv /tmp/kind /usr/local/bin/kind
fi

if ! command -v kubectl >/dev/null 2>&1; then
  echo "    Installing kubectl..."
  curl -Lo /tmp/kubectl "https://dl.k8s.io/release/$(curl -Ls https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  chmod +x /tmp/kubectl
  sudo mv /tmp/kubectl /usr/local/bin/kubectl
fi

echo "==> Creating kind cluster (${CLUSTER_NAME}) if it doesn't exist..."
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
  kind create cluster --name "${CLUSTER_NAME}"
else
  echo "    Cluster already exists, reusing it."
fi

echo "==> Building pulse-api image..."
docker build -t pulse-api:local -f api/common/Dockerfile api/common

echo "==> Loading image into kind..."
kind load docker-image pulse-api:local --name "${CLUSTER_NAME}"

echo "==> Applying mongo, redis, pulse-api..."
kubectl apply -f k8s/mongo.yaml -f k8s/redis.yaml -f k8s/api-deployment.yaml -f k8s/api-service.yaml

echo "==> Waiting for pulse-api to be ready..."
kubectl rollout status deployment/pulse-api --timeout=450s

echo "==> Installing metrics-server..."
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml


kubectl patch deployment metrics-server -n kube-system --type=json -p='[
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"},
  {"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--metric-resolution=15s"}
]'

echo "==> Waiting for metrics-server to be ready..."
kubectl rollout status deployment/metrics-server -n kube-system --timeout=120s

echo "==> Applying HPA and load-generator (starts at 0 replicas)..."
kubectl apply -f k8s/hpa.yaml -f k8s/load-generator.yaml

echo ""
echo "==> Ready. Useful commands:"
echo "    kubectl get hpa pulse-api -w                              # watch the HPA decide"
echo "    kubectl get pods -w                                       # watch pods come and go"
echo "    kubectl scale deployment/load-generator --replicas=6      # ramp up traffic"
echo "    kubectl scale deployment/load-generator --replicas=0      # stop traffic"
echo "    kind delete cluster --name ${CLUSTER_NAME}                # tear down when done"
