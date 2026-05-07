#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

WEB_SOURCE_DIR="${WEB_SOURCE_DIR:-${REPO_ROOT}/web}"
FRONTEND_ROOT="${FRONTEND_ROOT:-/var/www/fund-live}"
FRONTEND_RELEASE_ROOT="${FRONTEND_RELEASE_ROOT:-${FRONTEND_ROOT}/release}"
FRONTEND_CURRENT_LINK="${FRONTEND_CURRENT_LINK:-${FRONTEND_ROOT}/current}"
FRONTEND_PM2_NAME="${FRONTEND_PM2_NAME:-fund-live-frontend}"
FRONTEND_PORT="${FRONTEND_PORT:-13069}"
FRONTEND_HEALTHCHECK_URL="${FRONTEND_HEALTHCHECK_URL:-http://127.0.0.1:${FRONTEND_PORT}}"
FRONTEND_HEALTHCHECK_RETRIES="${FRONTEND_HEALTHCHECK_RETRIES:-20}"
FRONTEND_HEALTHCHECK_INTERVAL_SECONDS="${FRONTEND_HEALTHCHECK_INTERVAL_SECONDS:-2}"
FRONTEND_RELEASE_LABEL="${FRONTEND_RELEASE_LABEL:-$(date +%Y%m%d-%H%M%S)}"
FRONTEND_KEEP_RELEASES="${FRONTEND_KEEP_RELEASES:-5}"
FRONTEND_INSTALL_CMD="${FRONTEND_INSTALL_CMD:-npm ci}"
FRONTEND_BUILD_CMD="${FRONTEND_BUILD_CMD:-npm run build}"

RELEASE_DIR="${FRONTEND_RELEASE_ROOT}/${FRONTEND_RELEASE_LABEL}"

copy_source() {
  if command -v rsync >/dev/null 2>&1; then
    rsync -a \
      --delete \
      --exclude '.next' \
      --exclude 'node_modules' \
      --exclude '.git' \
      "${WEB_SOURCE_DIR}/" "${RELEASE_DIR}/"
    return
  fi

  tar \
    --exclude='.next' \
    --exclude='node_modules' \
    --exclude='.git' \
    -C "${WEB_SOURCE_DIR}" -cf - . | tar -C "${RELEASE_DIR}" -xf -
}

echo "[frontend] source dir: ${WEB_SOURCE_DIR}"
echo "[frontend] release dir: ${RELEASE_DIR}"

mkdir -p "${FRONTEND_RELEASE_ROOT}"
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"

copy_source

cd "${RELEASE_DIR}"
eval "${FRONTEND_INSTALL_CMD}"
eval "${FRONTEND_BUILD_CMD}"

ln -sfn "${RELEASE_DIR}" "${FRONTEND_CURRENT_LINK}"

if pm2 describe "${FRONTEND_PM2_NAME}" >/dev/null 2>&1; then
  echo "[frontend] restarting pm2 app: ${FRONTEND_PM2_NAME}"
  pm2 restart "${FRONTEND_PM2_NAME}" --update-env
else
  echo "[frontend] starting pm2 app: ${FRONTEND_PM2_NAME}"
  pm2 start npm --name "${FRONTEND_PM2_NAME}" --cwd "${FRONTEND_CURRENT_LINK}" -- run start -- -p "${FRONTEND_PORT}"
fi

pm2 ls

echo "[frontend] waiting for healthcheck: ${FRONTEND_HEALTHCHECK_URL}"
for ((i = 1; i <= FRONTEND_HEALTHCHECK_RETRIES; i++)); do
  if curl -fsS "${FRONTEND_HEALTHCHECK_URL}" >/dev/null; then
    echo "[frontend] healthcheck passed"
    break
  fi
  sleep "${FRONTEND_HEALTHCHECK_INTERVAL_SECONDS}"
  if [[ "${i}" -eq "${FRONTEND_HEALTHCHECK_RETRIES}" ]]; then
    echo "[frontend] healthcheck failed after ${FRONTEND_HEALTHCHECK_RETRIES} attempts"
    exit 1
  fi
done

if [[ "${FRONTEND_KEEP_RELEASES}" =~ ^[0-9]+$ ]] && (( FRONTEND_KEEP_RELEASES > 0 )); then
  mapfile -t RELEASE_DIRS < <(find "${FRONTEND_RELEASE_ROOT}" -mindepth 1 -maxdepth 1 -type d | sort)
  if (( ${#RELEASE_DIRS[@]} > FRONTEND_KEEP_RELEASES )); then
    TO_DELETE_COUNT=$((${#RELEASE_DIRS[@]} - FRONTEND_KEEP_RELEASES))
    for OLD_DIR in "${RELEASE_DIRS[@]:0:${TO_DELETE_COUNT}}"; do
      if [[ "${OLD_DIR}" != "${RELEASE_DIR}" ]]; then
        rm -rf "${OLD_DIR}"
      fi
    done
  fi
fi

echo "[frontend] deploy complete"
