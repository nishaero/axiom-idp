#!/usr/bin/env bash

set -euo pipefail

MODE=${MODE:-docker}
IMAGE_NAME=${IMAGE_NAME:-axiom-idp:e2e}
NAMESPACE=${NAMESPACE:-axiom}
MANIFEST=${MANIFEST:-deployments/k8s-deployment.yaml}

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PORT_FORWARD_PID=""

cleanup_docker() {
  docker compose down -v --remove-orphans >/dev/null 2>&1 || true
}

cleanup_minikube() {
  if [[ -n "${PORT_FORWARD_PID}" ]]; then
    kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
    wait "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
  fi
  kubectl delete namespace "${NAMESPACE}" >/dev/null 2>&1 || true
}

cleanup() {
  local exit_code=$?

  if [[ "${MODE}" == "docker" ]]; then
    cleanup_docker
  elif [[ "${MODE}" == "minikube" ]]; then
    cleanup_minikube
  fi

  exit "${exit_code}"
}

trap cleanup EXIT INT TERM

case "${MODE}" in
  docker)
    docker compose up -d --build
    "${ROOT_DIR}/verify-app.sh"
    ;;
  minikube)
    if ! command -v minikube >/dev/null 2>&1; then
      echo "minikube is required for MODE=minikube"
      exit 1
    fi

    if ! command -v kubectl >/dev/null 2>&1; then
      echo "kubectl is required for MODE=minikube"
      exit 1
    fi

    minikube status >/dev/null 2>&1 || minikube start --driver=docker

    docker build -t "${IMAGE_NAME}" "${ROOT_DIR}"
    minikube image load "${IMAGE_NAME}"

    kubectl apply -f "${ROOT_DIR}/${MANIFEST}"
    kubectl -n "${NAMESPACE}" create secret generic axiom-runtime-secrets \
      --from-literal=AXIOM_SESSION_SECRET="${AXIOM_SESSION_SECRET:-test-session-secret-for-ci}" \
      --dry-run=client -o yaml | kubectl apply -f -
    kubectl -n "${NAMESPACE}" set image deployment/axiom-server axiom-server="${IMAGE_NAME}"
    kubectl -n "${NAMESPACE}" rollout status deployment/axiom-server --timeout=180s

    kubectl -n "${NAMESPACE}" port-forward svc/axiom-server 8080:80 >/tmp/axiom-port-forward.log 2>&1 &
    PORT_FORWARD_PID=$!
    sleep 5
    BASE_URL=${BASE_URL:-http://127.0.0.1:8080} "${ROOT_DIR}/verify-app.sh"
    ;;
  *)
    echo "Unsupported MODE=${MODE}. Use docker or minikube."
    exit 1
    ;;
esac
