#!/usr/bin/env bash

set -euo pipefail

BASE_URL=${BASE_URL:-http://127.0.0.1:8080}
FRONTEND_URL=${FRONTEND_URL:-$BASE_URL}
AUTH_TOKEN=${AUTH_TOKEN:-}
TIMEOUT_SECONDS=${TIMEOUT_SECONDS:-60}

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

passed=0
failed=0

log() {
  printf '%b\n' "$*"
}

pass() {
  log "${GREEN}PASS${NC} $1"
  passed=$((passed + 1))
}

fail() {
  log "${RED}FAIL${NC} $1"
  failed=$((failed + 1))
}

wait_for_url() {
  local url=$1
  local deadline=$((SECONDS + TIMEOUT_SECONDS))

  until curl -fsS "$url" >/dev/null 2>&1; do
    if (( SECONDS >= deadline )); then
      return 1
    fi
    sleep 1
  done
}

wait_for_job_status() {
  local job_url=$1
  local deadline=$((SECONDS + TIMEOUT_SECONDS))
  local payload

  while (( SECONDS < deadline )); do
    payload=$(request GET "${BASE_URL}${job_url}")
    if printf '%s' "$payload" | grep -q '"status":"succeeded"'; then
      printf '%s' "$payload"
      return 0
    fi
    if printf '%s' "$payload" | grep -q '"status":"failed"'; then
      printf '%s' "$payload"
      return 1
    fi
    sleep 2
  done

  payload=$(request GET "${BASE_URL}${job_url}")
  printf '%s' "$payload"
  return 1
}

json_field() {
  local payload=$1
  local field=$2

  printf '%s' "$payload" | sed -n "s/.*\"${field}\":\"\([^\"]*\)\".*/\1/p"
}

request() {
  local method=$1
  local url=$2
  local body=${3:-}
  local headers=("-H" "Accept: application/json")

  if [[ -n "${AUTH_TOKEN}" ]]; then
    headers+=("-H" "Authorization: Bearer ${AUTH_TOKEN}")
  fi

  if [[ -n "${body}" ]]; then
    curl -fsS -X "$method" "${headers[@]}" -H "Content-Type: application/json" -d "$body" "$url"
  else
    curl -fsS -X "$method" "${headers[@]}" "$url"
  fi
}

json_has() {
  local payload=$1
  local needle=$2

  if printf '%s' "$payload" | grep -q "$needle"; then
    return 0
  fi
  return 1
}

log "Checking readiness endpoint at ${BASE_URL}/ready"
wait_for_url "${BASE_URL}/ready" || {
  fail "Readiness endpoint did not become ready"
  exit 1
}

health_payload=$(request GET "${BASE_URL}/ready")
if printf '%s' "$health_payload" | grep -q '"status":"ready"'; then
  pass "Readiness endpoint returns ready"
else
  fail "Unexpected readiness payload: ${health_payload}"
fi

log "Checking auth/login"
login_payload=$(request POST "${BASE_URL}/api/v1/auth/login" '{"user_id":"e2e-engineer","roles":["viewer","engineer"],"expires_in":"2h"}')
token=$(json_field "$login_payload" token)
if [[ -z "${AUTH_TOKEN}" && -n "${token}" ]]; then
  AUTH_TOKEN=$token
fi
if [[ -z "${AUTH_TOKEN}" ]]; then
  AUTH_TOKEN="bearer-check"
fi
pass "Login endpoint responds"

catalog_payload=$(request GET "${BASE_URL}/api/v1/catalog/services")
if printf '%s' "$catalog_payload" | grep -q '"services"'; then
  pass "Catalog services endpoint responds"
else
  fail "Catalog services payload: ${catalog_payload}"
fi

search_payload=$(request GET "${BASE_URL}/api/v1/catalog/search?query=service")
if printf '%s' "$search_payload" | grep -q '"results"'; then
  pass "Catalog search endpoint responds"
else
  fail "Catalog search payload: ${search_payload}"
fi

ai_payload=$(request POST "${BASE_URL}/api/v1/ai/query" '{"query":"List available services","context_limit":2000}')
if printf '%s' "$ai_payload" | grep -q '"response"'; then
  pass "AI query endpoint responds"
else
  fail "AI query payload: ${ai_payload}"
fi

if [[ -n "${E2E_DEPLOY_QUERY:-}" ]]; then
  deploy_payload=$(request POST "${BASE_URL}/api/v1/ai/query" "{\"query\":\"${E2E_DEPLOY_QUERY}\"}")
  if json_has "$deploy_payload" '"intent":"deployment_apply_argocd"' && json_has "$deploy_payload" '"delivery_provider":"argocd"' && json_has "$deploy_payload" '"github-argocd-kubernetes"'; then
    pass "Argo CD deploy request routed through GitHub and Argo CD"
  else
    fail "Unexpected Argo CD deploy payload: ${deploy_payload}"
  fi

  deploy_job_url=$(json_field "$deploy_payload" job_status_url)
  if [[ -n "$deploy_job_url" ]]; then
    deploy_job_payload=$(wait_for_job_status "$deploy_job_url") || {
      fail "Argo CD deployment job did not complete successfully: ${deploy_job_payload}"
      exit 1
    }
    if json_has "$deploy_job_payload" '"status":"succeeded"'; then
      pass "Argo CD deployment job completed"
    else
      fail "Argo CD deployment job did not succeed: ${deploy_job_payload}"
      exit 1
    fi
  fi

  if command -v kubectl >/dev/null 2>&1 && [[ -n "${E2E_DEPLOY_NAMESPACE:-}" && -n "${E2E_DEPLOY_NAME:-}" ]]; then
    kubectl -n "${E2E_DEPLOY_NAMESPACE}" wait --for=condition=Available "deployment/${E2E_DEPLOY_NAME}" --timeout=300s >/dev/null
    pass "Argo CD rollout created a live deployment"
  fi

  deploy_status_query=${E2E_DEPLOY_STATUS_QUERY:-"argocd deployment status ${E2E_DEPLOY_NAME:-demo-web} in namespace ${E2E_DEPLOY_NAMESPACE:-axiom-apps} from github"}
  deploy_status_payload=$(request POST "${BASE_URL}/api/v1/ai/query" "{\"query\":\"${deploy_status_query}\"}")
  if json_has "$deploy_status_payload" '"intent":"deployment_status_argocd"' && json_has "$deploy_status_payload" '"delivery_provider":"argocd"'; then
    pass "Argo CD deployment status query responds"
  else
    fail "Unexpected Argo CD status payload: ${deploy_status_payload}"
  fi
fi

if [[ -n "${E2E_INFRA_QUERY:-}" ]]; then
  infra_payload=$(request POST "${BASE_URL}/api/v1/ai/query" "{\"query\":\"${E2E_INFRA_QUERY}\"}")
  if json_has "$infra_payload" '"intent":"infrastructure_apply_terraform"' && json_has "$infra_payload" '"github-argocd-terraform-job"'; then
    pass "Terraform infrastructure request routed through GitHub and Argo CD"
  else
    fail "Unexpected Terraform infrastructure payload: ${infra_payload}"
  fi

  infra_job_url=$(json_field "$infra_payload" job_status_url)
  if [[ -n "$infra_job_url" ]]; then
    infra_job_payload=$(wait_for_job_status "$infra_job_url") || {
      fail "Terraform infrastructure job did not complete successfully: ${infra_job_payload}"
      exit 1
    }
    if json_has "$infra_job_payload" '"status":"succeeded"'; then
      pass "Terraform infrastructure job completed"
    else
      fail "Terraform infrastructure job did not succeed: ${infra_job_payload}"
      exit 1
    fi
  fi

  if command -v kubectl >/dev/null 2>&1 && [[ -n "${E2E_INFRA_NAMESPACE:-}" && -n "${E2E_INFRA_NAME:-}" ]]; then
    kubectl -n axiom-infra-jobs wait --for=condition=Complete "job/tf-apply-${E2E_INFRA_NAME}" --timeout=300s >/dev/null
    pass "Terraform job completed in cluster"
    kubectl -n "${E2E_INFRA_NAMESPACE}" get configmap terraform-footprint >/dev/null
    pass "Terraform created target namespace artifacts"
  fi
fi

observability_payload=$(request GET "${BASE_URL}/api/v1/platform/observability")
if printf '%s' "$observability_payload" | grep -q '"metrics_endpoint":"\/metrics"'; then
  pass "Observability snapshot responds"
else
  fail "Observability payload: ${observability_payload}"
fi

metrics_payload=$(request GET "${BASE_URL}/metrics")
if printf '%s' "$metrics_payload" | grep -q 'axiom_http_requests_total'; then
  pass "Prometheus metrics endpoint responds"
else
  fail "Metrics payload missing axiom_http_requests_total"
fi

front_payload=$(request GET "${FRONTEND_URL}/")
if printf '%s' "$front_payload" | grep -qi '<html\|axiom\|dashboard'; then
  pass "Frontend responds"
else
  fail "Frontend payload did not look like HTML"
fi

headers=$(curl -fsS -D - -o /dev/null "${BASE_URL}/ready")
if printf '%s' "$headers" | grep -qi 'x-frame-options\|x-content-type-options\|content-security-policy'; then
  pass "Security headers present"
else
  fail "Security headers missing"
fi

log ""
log "Summary: ${passed} passed, ${failed} failed"

if (( failed > 0 )); then
  exit 1
fi
