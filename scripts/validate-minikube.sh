#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
AXIOM_AI_BACKEND=${AXIOM_AI_BACKEND:-openai} \
AXIOM_AI_BASE_URL=${AXIOM_AI_BASE_URL:-${OPENAI_COMPATIBLE_BASE_URL:-http://host.minikube.internal:11434}} \
AXIOM_AI_API_KEY=${AXIOM_AI_API_KEY:-local-openai-compatible} \
AXIOM_AI_MODEL=${AXIOM_AI_MODEL:-${OPENAI_COMPATIBLE_MODEL:-qwen3.5:9b}} \
AXIOM_SESSION_SECRET=${AXIOM_SESSION_SECRET:-test-session-secret-for-ci} \
ARGOCD_NAMESPACE=${ARGOCD_NAMESPACE:-argocd} \
MODE=minikube "${ROOT_DIR}/test-e2e.sh"
