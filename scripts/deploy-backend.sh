#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

BACKEND_BINARY_RELATIVE_PATH="${BACKEND_BINARY_RELATIVE_PATH:-target/bin/fund-live-backend}"
BACKEND_BINARY_PATH="${BACKEND_BINARY_PATH:-${REPO_ROOT}/${BACKEND_BINARY_RELATIVE_PATH}}"
BACKEND_SERVICE_NAME="${BACKEND_SERVICE_NAME:-fund-live-backend}"
BACKEND_HEALTHCHECK_URL="${BACKEND_HEALTHCHECK_URL:-http://127.0.0.1:13896/health}"
BACKEND_HEALTHCHECK_RETRIES="${BACKEND_HEALTHCHECK_RETRIES:-20}"
BACKEND_HEALTHCHECK_INTERVAL_SECONDS="${BACKEND_HEALTHCHECK_INTERVAL_SECONDS:-2}"
BACKEND_CONFIG_PATH="${BACKEND_CONFIG_PATH:-/etc/fund-live/fundlive.yaml}"
BACKEND_INSTALL_SYSTEMD_CONFIG_OVERRIDE="${BACKEND_INSTALL_SYSTEMD_CONFIG_OVERRIDE:-true}"


ensure_systemd_config_override() {
  if [[ "${BACKEND_INSTALL_SYSTEMD_CONFIG_OVERRIDE}" != "true" ]]; then
    return
  fi

  if [[ ! -f "${BACKEND_CONFIG_PATH}" ]]; then
    echo "[backend] runtime config not found, skip systemd override: ${BACKEND_CONFIG_PATH}"
    return
  fi

  local override_dir="/etc/systemd/system/${BACKEND_SERVICE_NAME}.service.d"
  local override_file="${override_dir}/override.conf"

  echo "[backend] ensuring systemd config override: FUNDLIVE_CONFIG=${BACKEND_CONFIG_PATH}"
  mkdir -p "${override_dir}"
  cat > "${override_file}" <<EOF
[Service]
Environment=FUNDLIVE_CONFIG=${BACKEND_CONFIG_PATH}
EOF
  systemctl daemon-reload
}

echo "[backend] repo root: ${REPO_ROOT}"
echo "[backend] building binary: ${BACKEND_BINARY_PATH}"

mkdir -p "$(dirname "${BACKEND_BINARY_PATH}")"

cd "${REPO_ROOT}"
go build -o "${BACKEND_BINARY_PATH}" ./cmd/server

ensure_systemd_config_override

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
