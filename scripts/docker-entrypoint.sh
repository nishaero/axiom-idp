#!/usr/bin/env bash

set -euo pipefail

: "${AXIOM_HOST:=0.0.0.0}"
: "${AXIOM_PORT:=8081}"
: "${AXIOM_ENV:=production}"
: "${AXIOM_LOG_LEVEL:=info}"

export AXIOM_HOST AXIOM_PORT AXIOM_ENV AXIOM_LOG_LEVEL

BACKEND_LOG=${BACKEND_LOG:-/tmp/axiom-backend.log}
NGINX_LOG=${NGINX_LOG:-/tmp/axiom-nginx.log}

cleanup() {
  local exit_code=$?

  if [[ -n "${NGINX_PID:-}" ]] && kill -0 "${NGINX_PID}" 2>/dev/null; then
    kill "${NGINX_PID}" 2>/dev/null || true
  fi

  if [[ -n "${BACKEND_PID:-}" ]] && kill -0 "${BACKEND_PID}" 2>/dev/null; then
    kill "${BACKEND_PID}" 2>/dev/null || true
  fi

  wait "${NGINX_PID:-}" 2>/dev/null || true
  wait "${BACKEND_PID:-}" 2>/dev/null || true

  exit "${exit_code}"
}

trap cleanup EXIT INT TERM

/usr/local/bin/axiom-server >"${BACKEND_LOG}" 2>&1 &
BACKEND_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${AXIOM_PORT}/ready" >/dev/null 2>&1; then
    break
  fi

  if ! kill -0 "${BACKEND_PID}" 2>/dev/null; then
    echo "Backend exited before it became healthy."
    cat "${BACKEND_LOG}" || true
    exit 1
  fi

  sleep 1
done

if ! curl -fsS "http://127.0.0.1:${AXIOM_PORT}/ready" >/dev/null 2>&1; then
  echo "Backend did not become healthy in time."
  cat "${BACKEND_LOG}" || true
  exit 1
fi

nginx -g 'daemon off;' >"${NGINX_LOG}" 2>&1 &
NGINX_PID=$!

wait -n "${BACKEND_PID}" "${NGINX_PID}"
