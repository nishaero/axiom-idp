#!/usr/bin/env bash

set -euo pipefail

MODE=${MODE:-docker}
IMAGE_NAME=${IMAGE_NAME:-axiom-idp:e2e-$(date -u +%Y%m%d%H%M%S)}
NAMESPACE=${NAMESPACE:-axiom}
MANIFEST=${MANIFEST:-deployments/k8s-deployment.yaml}
OPENAI_COMPATIBLE_BASE_URL=${OPENAI_COMPATIBLE_BASE_URL:-${OLLAMA_BASE_URL:-http://host.minikube.internal:11434}}
OPENAI_COMPATIBLE_MODEL=${OPENAI_COMPATIBLE_MODEL:-${OLLAMA_MODEL:-qwen3.5:9b}}
ARGOCD_NAMESPACE=${ARGOCD_NAMESPACE:-argocd}
ARGOCD_MANIFEST_URL=${ARGOCD_MANIFEST_URL:-https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml}
GITHUB_TOKEN=${GITHUB_TOKEN:-$(gh auth token 2>/dev/null || true)}

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PORT_FORWARD_PID=""
PORT_FORWARD_LOCAL_PORT=${PORT_FORWARD_LOCAL_PORT:-18080}

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

ensure_argocd() {
  if ! kubectl get namespace "${ARGOCD_NAMESPACE}" >/dev/null 2>&1; then
    kubectl create namespace "${ARGOCD_NAMESPACE}" >/dev/null
  fi

  if ! kubectl -n "${ARGOCD_NAMESPACE}" get deployment argocd-server >/dev/null 2>&1; then
    kubectl apply -n "${ARGOCD_NAMESPACE}" --server-side --force-conflicts -f "${ARGOCD_MANIFEST_URL}"
  fi

  kubectl -n "${ARGOCD_NAMESPACE}" rollout status deployment/argocd-server --timeout=300s
  kubectl -n "${ARGOCD_NAMESPACE}" rollout status deployment/argocd-repo-server --timeout=300s
  kubectl -n "${ARGOCD_NAMESPACE}" rollout status statefulset/argocd-application-controller --timeout=300s

  kubectl apply --server-side --force-conflicts -f - <<EOF >/dev/null
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argocd-application-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: argocd-application-controller
subjects:
- kind: ServiceAccount
  name: argocd-application-controller
  namespace: ${ARGOCD_NAMESPACE}
EOF

  kubectl apply --server-side --force-conflicts -f - <<EOF >/dev/null
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argocd-applicationset-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: argocd-applicationset-controller
subjects:
- kind: ServiceAccount
  name: argocd-applicationset-controller
  namespace: ${ARGOCD_NAMESPACE}
EOF

  kubectl apply --server-side --force-conflicts -f - <<EOF >/dev/null
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argocd-server
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: argocd-server
subjects:
- kind: ServiceAccount
  name: argocd-server
  namespace: ${ARGOCD_NAMESPACE}
EOF
}

configure_axiom_runtime() {
  if [[ -z "${GITHUB_TOKEN}" ]]; then
    echo "A GitHub token is required for the Argo CD validation path"
    exit 1
  fi

  kubectl -n "${NAMESPACE}" set env deployment/axiom-server \
    AXIOM_AI_BACKEND="${AXIOM_AI_BACKEND:-openai}" \
    AXIOM_AI_BASE_URL="${OPENAI_COMPATIBLE_BASE_URL}" \
    AXIOM_AI_API_KEY="${AXIOM_AI_API_KEY:-local-openai-compatible}" \
    AXIOM_AI_MODEL="${OPENAI_COMPATIBLE_MODEL}" \
    AXIOM_GITOPS_REPO_URL="https://github.com/nishaero/axiom-idp.git" \
    AXIOM_ARGOCD_NAMESPACE="${ARGOCD_NAMESPACE}" \
    GIT_CONFIG_COUNT=1 \
    "GIT_CONFIG_KEY_0=url.https://x-access-token:${GITHUB_TOKEN}@github.com/.insteadOf" \
    GIT_CONFIG_VALUE_0="https://github.com/" >/dev/null
}

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

    if ! command -v gh >/dev/null 2>&1 && [[ -z "${GITHUB_TOKEN}" ]]; then
      echo "gh CLI or GITHUB_TOKEN is required for the GitHub-backed validation path"
      exit 1
    fi

    minikube status >/dev/null 2>&1 || minikube start --driver=docker

    ensure_argocd

    docker build -t "${IMAGE_NAME}" "${ROOT_DIR}"
    minikube image load "${IMAGE_NAME}"

    kubectl apply -f "${ROOT_DIR}/${MANIFEST}"
    kubectl -n "${NAMESPACE}" create secret generic axiom-runtime-secrets \
      --from-literal=AXIOM_SESSION_SECRET="${AXIOM_SESSION_SECRET:-test-session-secret-for-ci}" \
      --dry-run=client -o yaml | kubectl apply -f -
    configure_axiom_runtime
    kubectl -n "${NAMESPACE}" set image deployment/axiom-server axiom-server="${IMAGE_NAME}"
    kubectl -n "${NAMESPACE}" rollout status deployment/axiom-server --timeout=180s
    kubectl -n "${NAMESPACE}" wait --for=condition=Ready pod -l app.kubernetes.io/name=axiom-server --timeout=120s >/dev/null

    pod_name=$(
      kubectl -n "${NAMESPACE}" get pods \
        -l app.kubernetes.io/name=axiom-server \
        --field-selector=status.phase=Running \
        --sort-by=.metadata.creationTimestamp \
        -o jsonpath='{.items[-1:].metadata.name}'
    )
    if [[ -z "${pod_name}" ]]; then
      echo "Failed to determine a ready axiom-server pod for port-forwarding"
      exit 1
    fi

    kubectl -n "${NAMESPACE}" port-forward "pod/${pod_name}" "${PORT_FORWARD_LOCAL_PORT}:8080" >/tmp/axiom-port-forward.log 2>&1 &
    PORT_FORWARD_PID=$!
    sleep 5
    E2E_DEPLOY_QUERY=${E2E_DEPLOY_QUERY:-"deploy app demo-web using nginx:1.27-alpine via argocd from github in namespace axiom-apps"} \
    E2E_DEPLOY_NAME=${E2E_DEPLOY_NAME:-demo-web} \
    E2E_DEPLOY_NAMESPACE=${E2E_DEPLOY_NAMESPACE:-axiom-apps} \
    E2E_INFRA_QUERY=${E2E_INFRA_QUERY:-"provision infrastructure namespace platform-lab using terraform"} \
    E2E_INFRA_NAME=${E2E_INFRA_NAME:-platform-lab} \
    E2E_INFRA_NAMESPACE=${E2E_INFRA_NAMESPACE:-platform-lab} \
    BASE_URL=${BASE_URL:-http://127.0.0.1:${PORT_FORWARD_LOCAL_PORT}} "${ROOT_DIR}/verify-app.sh"
    ;;
  *)
    echo "Unsupported MODE=${MODE}. Use docker or minikube."
    exit 1
    ;;
esac
