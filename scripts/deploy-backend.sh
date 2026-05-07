#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BACKEND_BINARY_RELATIVE_PATH="${BACKEND_BINARY_RELATIVE_PATH:-target/bin/fund-live-backend}"
BACKEND_BINARY_PATH="${BACKEND_BINARY_PATH:-${REPO_ROOT}/${BACKEND_BINARY_RELATIVE_PATH}}"
BACKEND_SERVICE_NAME="${BACKEND_SERVICE_NAME:-fund-live-backend}"
BACKEND_HEALTHCHECK_URL="${BACKEND_HEALTHCHECK_URL:-http://127.0.0.1:8080/health}"
BACKEND_HEALTHCHECK_RETRIES="${BACKEND_HEALTHCHECK_RETRIES:-20}"
BACKEND_HEALTHCHECK_INTERVAL_SECONDS="${BACKEND_HEALTHCHECK_INTERVAL_SECONDS:-2}"

echo "[backend] repo root: ${REPO_ROOT}"
echo "[backend] building binary: ${BACKEND_BINARY_PATH}"

mkdir -p "$(dirname "${BACKEND_BINARY_PATH}")"

cd "${REPO_ROOT}"
go build -o "${BACKEND_BINARY_PATH}" ./cmd/server

echo "[backend] restarting systemd service: ${BACKEND_SERVICE_NAME}"
systemctl restart "${BACKEND_SERVICE_NAME}"
systemctl is-active --quiet "${BACKEND_SERVICE_NAME}"

echo "[backend] waiting for healthcheck: ${BACKEND_HEALTHCHECK_URL}"
for ((i = 1; i <= BACKEND_HEALTHCHECK_RETRIES; i++)); do
  if curl -fsS "${BACKEND_HEALTHCHECK_URL}" >/dev/null; then
    echo "[backend] healthcheck passed"
    exit 0
  fi
  sleep "${BACKEND_HEALTHCHECK_INTERVAL_SECONDS}"
done

echo "[backend] healthcheck failed after ${BACKEND_HEALTHCHECK_RETRIES} attempts"
journalctl -u "${BACKEND_SERVICE_NAME}" -n 100 --no-pager || true
exit 1
