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
login_payload=$(request POST "${BASE_URL}/api/v1/auth/login" '{}')
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

front_payload=$(request GET "${FRONTEND_URL}/")
if printf '%s' "$front_payload" | grep -qi '<html\|axiom\|dashboard'; then
  pass "Frontend responds"
else
  fail "Frontend payload did not look like HTML"
fi

headers=$(curl -fsS -D - -o /dev/null "${BASE_URL}/health")
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
