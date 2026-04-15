#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEPLOY_DIR="$ROOT_DIR/deploy"
ENV_FILE_PATH="$ROOT_DIR/.env.local"
ENV_TEMPLATE_PATH="$ROOT_DIR/.env.local.example"
LOG_DIR="$ROOT_DIR/logs"
RUNTIME_DIR="$LOG_DIR/runtime"
COLLECTED_DIR="$LOG_DIR/collected"
FRONT_LOG_DIR="$LOG_DIR/front-api"
USER_LOG_DIR="$LOG_DIR/user-rpc"
CONTENT_LOG_DIR="$LOG_DIR/content-rpc"
INTERACTION_LOG_DIR="$LOG_DIR/interaction-rpc"
COUNT_LOG_DIR="$LOG_DIR/count-rpc"
SEARCH_LOG_DIR="$LOG_DIR/search-rpc"

USER_RPC_PID_FILE="$RUNTIME_DIR/user-rpc.pid"
CONTENT_RPC_PID_FILE="$RUNTIME_DIR/content-rpc.pid"
INTERACTION_RPC_PID_FILE="$RUNTIME_DIR/interaction-rpc.pid"
FRONT_API_PID_FILE="$RUNTIME_DIR/front-api.pid"
COUNT_RPC_PID_FILE="$RUNTIME_DIR/count-rpc.pid"
SEARCH_RPC_PID_FILE="$RUNTIME_DIR/search-rpc.pid"

fct_require_env_file() {
  if [ -f "$ENV_FILE_PATH" ]; then
    :
  else
    if [ ! -f "$ENV_TEMPLATE_PATH" ]; then
      echo "Missing env template: $ENV_TEMPLATE_PATH" >&2
      exit 1
    fi

    cp "$ENV_TEMPLATE_PATH" "$ENV_FILE_PATH"
    echo "Created $ENV_FILE_PATH from template. Review values if your local ports differ."
  fi

  local line
  local key
  while IFS= read -r line || [ -n "$line" ]; do
    case "$line" in
      ""|\#*)
        continue
        ;;
    esac

    key=${line%%=*}
    if ! grep -q "^${key}=" "$ENV_FILE_PATH"; then
      printf '%s\n' "$line" >>"$ENV_FILE_PATH"
    fi
  done <"$ENV_TEMPLATE_PATH"

  local tmp_env
  tmp_env=$(mktemp)
  awk '
    /^[[:space:]]*#/ || /^[[:space:]]*$/ {
      print
      next
    }
    {
      split($0, pair, "=")
      key = pair[1]
      if (seen[key]++) {
        next
      }
      print
    }
  ' "$ENV_FILE_PATH" >"$tmp_env"
  mv "$tmp_env" "$ENV_FILE_PATH"
}

fct_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    (
      cd "$DEPLOY_DIR"
      docker compose --env-file .env -f docker-compose.yml "$@"
    )
    return 0
  fi

  if ! command -v powershell.exe >/dev/null 2>&1; then
    echo "docker compose is unavailable and powershell.exe is missing." >&2
    return 1
  fi

  local deploy_dir_win
  deploy_dir_win=$(wslpath -w "$DEPLOY_DIR")

  local ps_cmd='cmd /c "pushd '"$deploy_dir_win"' && docker compose --env-file .env -f docker-compose.yml'
  local arg
  for arg in "$@"; do
    ps_cmd="$ps_cmd $arg"
  done
  ps_cmd="$ps_cmd\""

  powershell.exe -NoProfile -Command "$ps_cmd"
}

fct_port_from_listen_on() {
  local listen_on="$1"
  printf '%s\n' "${listen_on##*:}"
}

fct_port_from_url() {
  local url="$1"
  printf '%s\n' "$url" | sed -E 's#^https?://[^:]+:([0-9]+).*$#\1#'
}

fct_wait_for_port() {
  local port="$1"
  local name="$2"

  for _ in $(seq 1 120); do
    if (echo >"/dev/tcp/127.0.0.1/$port") >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for $name on 127.0.0.1:$port" >&2
  return 1
}

fct_stop_pid_file() {
  local pid_file="$1"

  if [ ! -f "$pid_file" ]; then
    return 0
  fi

  local pid
  pid=$(cat "$pid_file")
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
  rm -f "$pid_file"
}

mkdir -p \
  "$LOG_DIR" \
  "$RUNTIME_DIR" \
  "$COLLECTED_DIR" \
  "$FRONT_LOG_DIR" \
  "$USER_LOG_DIR" \
  "$CONTENT_LOG_DIR" \
  "$INTERACTION_LOG_DIR" \
  "$COUNT_LOG_DIR" \
  "$SEARCH_LOG_DIR"

fct_require_env_file
. "$ENV_FILE_PATH"
export ENV_FILE="$ENV_FILE_PATH"

BACKEND_RUNTIME="${BACKEND_RUNTIME:-docker}"
if [ "$BACKEND_RUNTIME" != "docker" ]; then
  echo "Unsupported BACKEND_RUNTIME=$BACKEND_RUNTIME. Day23 backend startup now expects docker." >&2
  exit 1
fi

USER_RPC_PORT=$(fct_port_from_listen_on "$USER_RPC_LISTEN_ON")
CONTENT_RPC_PORT=$(fct_port_from_listen_on "$CONTENT_RPC_LISTEN_ON")
INTERACTION_RPC_PORT=$(fct_port_from_listen_on "$INTERACTION_RPC_LISTEN_ON")
COUNT_RPC_PORT=$(fct_port_from_listen_on "$COUNT_RPC_LISTEN_ON")
SEARCH_RPC_PORT=$(fct_port_from_listen_on "$SEARCH_RPC_LISTEN_ON")
XXL_EXECUTOR_BIND_PORT=$(fct_port_from_listen_on "$XXL_EXECUTOR_ADDRESS")
XXL_ADMIN_PORT=$(fct_port_from_url "$XXL_JOB_ADMIN_ADDR")
KAFKA_PORT=$(fct_port_from_listen_on "$KAFKA_BROKERS")
FRONT_PROM_PORT="${PROM_PORT}"
CONTENT_PROM_PORT="${CONTENT_PROM_PORT}"
INTERACTION_PROM_PORT="${INTERACTION_PROM_PORT}"
COUNT_PROM_PORT="${COUNT_PROM_PORT}"
USER_PROM_PORT="${USER_PROM_PORT}"
SEARCH_PROM_PORT="${SEARCH_PROM_PORT}"
PROMETHEUS_HOST_PORT="${PROMETHEUS_HOST_PORT}"
GATEWAY_HOST_PORT="${GATEWAY_HOST_PORT:-18080}"
ENABLE_LOG_PIPELINE="${ENABLE_LOG_PIPELINE:-0}"
ENABLE_TRACE_PIPELINE="${ENABLE_TRACE_PIPELINE:-1}"
OTEL_COLLECTOR_GRPC_HOST_PORT="${OTEL_COLLECTOR_GRPC_HOST_PORT:-4317}"
OTEL_COLLECTOR_HTTP_HOST_PORT="${OTEL_COLLECTOR_HTTP_HOST_PORT:-4318}"
JAEGER_HOST_PORT="${JAEGER_HOST_PORT:-16686}"

fct_stop_pid_file "$USER_RPC_PID_FILE"
fct_stop_pid_file "$CONTENT_RPC_PID_FILE"
fct_stop_pid_file "$INTERACTION_RPC_PID_FILE"
fct_stop_pid_file "$COUNT_RPC_PID_FILE"
fct_stop_pid_file "$SEARCH_RPC_PID_FILE"
fct_stop_pid_file "$FRONT_API_PID_FILE"

infra_services=(etcd redis mysql kafka canal xxl-job-admin prometheus)
backend_services=(user-rpc content-rpc interaction-rpc count-rpc search-rpc front-api)
delivery_services=(front-web nginx)

if [ "$ENABLE_LOG_PIPELINE" = "1" ]; then
  infra_services+=(logstash filebeat)
fi
if [ "$ENABLE_TRACE_PIPELINE" = "1" ]; then
  infra_services+=(jaeger otel-collector)
fi

echo "Starting zfeed Docker stack via Docker Compose..."
fct_docker_compose up -d --build "${infra_services[@]}" "${backend_services[@]}" "${delivery_services[@]}"

echo "Waiting for infrastructure ports..."
fct_wait_for_port "$ETCD_PORT" "etcd"
fct_wait_for_port "$REDIS_PORT" "redis"
fct_wait_for_port "$MYSQL_APP_PORT" "mysql"
fct_wait_for_port "$KAFKA_PORT" "kafka"
if [ -n "${XXL_ADMIN_PORT}" ] && [ "${XXL_ADMIN_PORT}" != "${XXL_JOB_ADMIN_ADDR}" ]; then
  fct_wait_for_port "$XXL_ADMIN_PORT" "xxl-job-admin"
fi
if [ -n "${PROMETHEUS_HOST_PORT}" ]; then
  fct_wait_for_port "$PROMETHEUS_HOST_PORT" "prometheus"
fi
if [ "$ENABLE_TRACE_PIPELINE" = "1" ]; then
  fct_wait_for_port "$OTEL_COLLECTOR_GRPC_HOST_PORT" "otel-collector grpc"
  fct_wait_for_port "$OTEL_COLLECTOR_HTTP_HOST_PORT" "otel-collector http"
  fct_wait_for_port "$JAEGER_HOST_PORT" "jaeger"
fi

echo "Waiting for backend ports..."
fct_wait_for_port "$USER_RPC_PORT" "user-rpc"
fct_wait_for_port "$USER_PROM_PORT" "user-rpc metrics"
fct_wait_for_port "$CONTENT_RPC_PORT" "content-rpc"
fct_wait_for_port "$CONTENT_PROM_PORT" "content-rpc metrics"
if [ -n "${XXL_EXECUTOR_BIND_PORT}" ] && [ "$XXL_EXECUTOR_BIND_PORT" != "$CONTENT_RPC_PORT" ]; then
  fct_wait_for_port "$XXL_EXECUTOR_BIND_PORT" "content-rpc xxl executor"
fi
fct_wait_for_port "$INTERACTION_RPC_PORT" "interaction-rpc"
fct_wait_for_port "$INTERACTION_PROM_PORT" "interaction-rpc metrics"
fct_wait_for_port "$COUNT_RPC_PORT" "count-rpc"
fct_wait_for_port "$COUNT_PROM_PORT" "count-rpc metrics"
fct_wait_for_port "$SEARCH_RPC_PORT" "search-rpc"
fct_wait_for_port "$SEARCH_PROM_PORT" "search-rpc metrics"
fct_wait_for_port "$FRONT_API_PORT" "front-api"
fct_wait_for_port "$FRONT_PROM_PORT" "front-api metrics"
fct_wait_for_port "$GATEWAY_HOST_PORT" "nginx gateway"

echo "zfeed docker stack is ready."
echo "  backend runtime: $BACKEND_RUNTIME"
echo "  compose file: $DEPLOY_DIR/docker-compose.yml"
echo "  host logs: $LOG_DIR"
echo "  service log roots: $FRONT_LOG_DIR $USER_LOG_DIR $CONTENT_LOG_DIR $INTERACTION_LOG_DIR $COUNT_LOG_DIR $SEARCH_LOG_DIR"
echo "  collected logs: $COLLECTED_DIR"
echo "  Web: http://127.0.0.1:$GATEWAY_HOST_PORT"
echo "  API direct: http://127.0.0.1:$FRONT_API_PORT"
echo "  metrics endpoints:"
echo "    front-api: http://127.0.0.1:$FRONT_PROM_PORT/metrics"
echo "    content-rpc: http://127.0.0.1:$CONTENT_PROM_PORT/metrics"
echo "    interaction-rpc: http://127.0.0.1:$INTERACTION_PROM_PORT/metrics"
echo "    count-rpc: http://127.0.0.1:$COUNT_PROM_PORT/metrics"
echo "    user-rpc: http://127.0.0.1:$USER_PROM_PORT/metrics"
echo "    search-rpc: http://127.0.0.1:$SEARCH_PROM_PORT/metrics"
echo "  prometheus: http://127.0.0.1:$PROMETHEUS_HOST_PORT"
echo "  log pipeline enabled: $ENABLE_LOG_PIPELINE"
echo "  trace pipeline enabled: $ENABLE_TRACE_PIPELINE"
if [ "$ENABLE_TRACE_PIPELINE" = "1" ]; then
  echo "  otel grpc endpoint: ${OTEL_ENDPOINT}"
  echo "  otel http endpoint: ${OTEL_HTTP_ENDPOINT}"
  echo "  jaeger: http://127.0.0.1:$JAEGER_HOST_PORT"
fi
echo "  xxl-job admin: http://127.0.0.1:$XXL_ADMIN_PORT/xxl-job-admin"
echo "  observability verify: $ROOT_DIR/script/test_observability.sh"
echo "  count write-chain verify: $ROOT_DIR/script/test_count_chain.sh"
echo "  count read-path verify: $ROOT_DIR/script/test_count_read_path.sh"
echo "  stop command: $ROOT_DIR/script/stop.sh"
