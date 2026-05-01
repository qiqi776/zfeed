#!/usr/bin/env bash
set -Eeuo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly DEPLOY_DIR="${ROOT_DIR}/deploy"
readonly COMPOSE_ENV_PATH="${DEPLOY_DIR}/.env"
readonly REBUILD="${REBUILD:-0}"
readonly RESEED_MYSQL="${RESEED_MYSQL:-0}"
readonly RESET_CANAL="${RESET_CANAL:-0}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found" >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose not available" >&2
  exit 1
fi

if [ ! -f "${COMPOSE_ENV_PATH}" ]; then
  echo "Missing compose env file: ${COMPOSE_ENV_PATH}" >&2
  exit 1
fi

# shellcheck disable=SC1090
set -a
. "${COMPOSE_ENV_PATH}"
set +a

cd "${DEPLOY_DIR}"
export RESEED_MYSQL RESET_CANAL

readonly ENABLE_LOG_PIPELINE_VALUE="${ENABLE_LOG_PIPELINE:-0}"
readonly ENABLE_TRACE_PIPELINE_VALUE="${ENABLE_TRACE_PIPELINE:-0}"

app_services=(front-api user-rpc content-rpc interaction-rpc count-rpc search-rpc)
app_images=(
  "${FRONT_API_IMAGE}"
  "${USER_RPC_IMAGE}"
  "${CONTENT_RPC_IMAGE}"
  "${INTERACTION_RPC_IMAGE}"
  "${COUNT_RPC_IMAGE}"
  "${SEARCH_RPC_IMAGE}"
)
services=(
  etcd
  redis
  mysql
  kafka
  canal
  xxl-job-admin
  prometheus
  "${app_services[@]}"
  nginx
)

if [ "${ENABLE_LOG_PIPELINE_VALUE}" = "1" ]; then
  services+=(logstash filebeat)
fi
if [ "${ENABLE_TRACE_PIPELINE_VALUE}" = "1" ]; then
  services+=(jaeger otel-collector)
fi

missing_services=()
for index in "${!app_services[@]}"; do
  if ! docker image inspect "${app_images[${index}]}" >/dev/null 2>&1; then
    missing_services+=("${app_services[${index}]}")
  fi
done

if [ "${REBUILD}" = "1" ]; then
  echo "Building application images..."
  docker compose --env-file .env -f docker-compose.yml build "${app_services[@]}"
elif [ "${#missing_services[@]}" -gt 0 ]; then
  echo "Building missing application images: ${missing_services[*]}"
  docker compose --env-file .env -f docker-compose.yml build "${missing_services[@]}"
fi

echo "Starting zfeed Docker stack..."
docker compose --env-file .env -f docker-compose.yml up -d "${services[@]}"

printf 'zfeed docker stack is starting.\n'
printf '  Gateway (/v1/*): http://127.0.0.1:%s\n' "${GATEWAY_HOST_PORT:-18080}"
printf '  API direct: http://127.0.0.1:%s\n' "${FRONT_API_PORT:-5000}"
printf '  stop command: bash ./script/stop.sh\n'
printf '  full rebuild: REBUILD=1 RESEED_MYSQL=1 RESET_CANAL=1 bash ./script/start.sh\n'
