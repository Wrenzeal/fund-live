#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

LEAN_IMAGE="${LEAN_IMAGE:-fund-lean-worker:latest}"
LEAN_CONTAINER_NAME="${LEAN_CONTAINER_NAME:-fundlive_lean_worker}"
LEAN_DOCKER_NETWORK="${LEAN_DOCKER_NETWORK:-fund_default}"
LEAN_JOB_VOLUME="${LEAN_JOB_VOLUME:-fund_fundlive_lean_jobs}"
LEAN_CONFIG_PATH="${LEAN_CONFIG_PATH:-/etc/fund-live/fundlive.yaml}"
LEAN_ENV_PATH="${LEAN_ENV_PATH:-/etc/fund-live/fundlive.env}"
LEAN_REDIS_URL="${LEAN_REDIS_URL:-redis://cache:6379/0}"
LEAN_DB_HOST="${LEAN_DB_HOST:-host.docker.internal}"
LEAN_READY_RETRIES="${LEAN_READY_RETRIES:-30}"
LEAN_READY_INTERVAL_SECONDS="${LEAN_READY_INTERVAL_SECONDS:-2}"

if [[ ! -f "${LEAN_CONFIG_PATH}" ]]; then
  echo "[lean-worker] runtime config not found: ${LEAN_CONFIG_PATH}" >&2
  exit 1
fi

if [[ -f "${LEAN_ENV_PATH}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${LEAN_ENV_PATH}"
  set +a
fi

LEAN_REDIS_KEY_PREFIX="${LEAN_REDIS_KEY_PREFIX:-${FUNDLIVE_REDIS_KEY_PREFIX:-fundlive}}"

if ! docker network inspect "${LEAN_DOCKER_NETWORK}" >/dev/null 2>&1; then
  echo "[lean-worker] Docker network not found: ${LEAN_DOCKER_NETWORK}" >&2
  exit 1
fi

echo "[lean-worker] building image: ${LEAN_IMAGE}"
docker build -t "${LEAN_IMAGE}" -f "${REPO_ROOT}/quant/lean-worker/Dockerfile" "${REPO_ROOT}"
docker volume create "${LEAN_JOB_VOLUME}" >/dev/null

candidate="${LEAN_CONTAINER_NAME}_candidate_$$"
cleanup_candidate() {
  if docker container inspect "${candidate}" >/dev/null 2>&1; then
    docker stop --time 10 "${candidate}" >/dev/null 2>&1 || true
    docker rm "${candidate}" >/dev/null 2>&1 || true
  fi
}
trap cleanup_candidate EXIT

echo "[lean-worker] starting deployment candidate"
docker run -d \
  --name "${candidate}" \
  --restart unless-stopped \
  --network "${LEAN_DOCKER_NETWORK}" \
  --add-host host.docker.internal:host-gateway \
  --security-opt no-new-privileges:true \
  --label fundlive.role=lean-worker \
  -e FUNDLIVE_CONFIG="${LEAN_CONFIG_PATH}" \
  -e DB_HOST="${LEAN_DB_HOST}" \
  -e REDIS_URL="${LEAN_REDIS_URL}" \
  -e REDIS_KEY_PREFIX="${LEAN_REDIS_KEY_PREFIX}" \
  -e LEAN_WORKER_GROUP="${LEAN_WORKER_GROUP:-lean-workers}" \
  -e LEAN_JOB_TIMEOUT_MINUTES="${LEAN_JOB_TIMEOUT_MINUTES:-30}" \
  -v "${LEAN_CONFIG_PATH}:${LEAN_CONFIG_PATH}:ro" \
  -v "${LEAN_JOB_VOLUME}:/var/lib/fundlive/lean-jobs" \
  "${LEAN_IMAGE}" >/dev/null

for ((i = 1; i <= LEAN_READY_RETRIES; i++)); do
  state="$(docker inspect --format '{{.State.Status}}' "${candidate}")"
  if [[ "${state}" == "running" ]] && docker logs "${candidate}" 2>&1 | grep -q "Lean worker ready"; then
    break
  fi
  if [[ "${state}" == "exited" || "${state}" == "dead" ]]; then
    echo "[lean-worker] candidate exited before becoming ready" >&2
    docker logs "${candidate}" >&2 || true
    exit 1
  fi
  if [[ "${i}" -eq "${LEAN_READY_RETRIES}" ]]; then
    echo "[lean-worker] readiness timed out" >&2
    docker logs "${candidate}" >&2 || true
    exit 1
  fi
  sleep "${LEAN_READY_INTERVAL_SECONDS}"
done

if docker container inspect "${LEAN_CONTAINER_NAME}" >/dev/null 2>&1; then
  echo "[lean-worker] stopping previous container"
  docker stop --time 30 "${LEAN_CONTAINER_NAME}" >/dev/null
  docker rm "${LEAN_CONTAINER_NAME}" >/dev/null
fi

docker rename "${candidate}" "${LEAN_CONTAINER_NAME}"
trap - EXIT

echo "[lean-worker] deployed"
docker ps --filter "name=^/${LEAN_CONTAINER_NAME}$" --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
