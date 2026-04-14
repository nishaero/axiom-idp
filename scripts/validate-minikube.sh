#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
AXIOM_AI_BACKEND=${AXIOM_AI_BACKEND:-local} MODE=minikube "${ROOT_DIR}/test-e2e.sh"
