#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly ROOT_DIR
readonly DEPLOY_DIR="${ROOT_DIR}/deploy"
readonly COMPOSE_ENV_PATH="${DEPLOY_DIR}/.env"
readonly REBUILD="${REBUILD:-0}"
readonly RESEED_MYSQL="${RESEED_MYSQL:-0}"
readonly RESET_CANAL="${RESET_CANAL:-0}"

ENV_OVERRIDE_ENABLE_LOG_PIPELINE_SET="${ENABLE_LOG_PIPELINE+x}"
ENV_OVERRIDE_ENABLE_LOG_PIPELINE="${ENABLE_LOG_PIPELINE:-}"
ENV_OVERRIDE_ENABLE_TRACE_PIPELINE_SET="${ENABLE_TRACE_PIPELINE+x}"
ENV_OVERRIDE_ENABLE_TRACE_PIPELINE="${ENABLE_TRACE_PIPELINE:-}"
ENV_OVERRIDE_ENABLE_GRAFANA_SET="${ENABLE_GRAFANA+x}"
ENV_OVERRIDE_ENABLE_GRAFANA="${ENABLE_GRAFANA:-}"
ENV_OVERRIDE_XXL_JOB_PORT_SET="${XXL_JOB_PORT+x}"
ENV_OVERRIDE_XXL_JOB_PORT="${XXL_JOB_PORT:-}"

fct_compose_owned_ports() {
  docker ps --filter "label=com.docker.compose.project=${COMPOSE_PROJECT_NAME}" --format '{{.Ports}}' 2>/dev/null | grep -oE ':[0-9]+->' | tr -cd '0-9\n' || true
}

fct_expected_port_rows() {
  cat <<EOF
nginx 网关|${GATEWAY_HOST_PORT}|80|默认 HTTP 压测入口
front-api|${FRONT_API_PORT}|5000|HTTP 直连入口
front-api Prometheus|${PROM_PORT}|9290|API 指标
content-rpc|5001|5001|内容服务 gRPC
content-rpc Prometheus|${CONTENT_PROM_PORT}|9291|内容服务指标
content-rpc XXL 执行器|${XXL_JOB_EXECUTOR_PORT}|${XXL_EXECUTOR_PORT}|XXL-JOB 回调执行器
interaction-rpc|5002|5002|互动服务 gRPC
interaction-rpc Prometheus|${INTERACTION_PROM_PORT}|9293|互动服务指标
user-rpc|5003|5003|用户服务 gRPC
user-rpc Prometheus|${USER_PROM_PORT}|9294|用户服务指标
count-rpc|5004|5004|计数服务 gRPC
count-rpc Prometheus|${COUNT_PROM_PORT}|9292|计数服务指标
search-rpc|5006|5006|搜索服务 gRPC
search-rpc Prometheus|${SEARCH_PROM_PORT}|9295|搜索服务指标
OpenSearch|${SEARCH_INDEX_ENGINE_HOST_PORT}|9200|搜索索引引擎
Prometheus|${PROMETHEUS_HOST_PORT}|9090|指标查询
MySQL|${MYSQL_PORT}|3306|主存储
Redis|${REDIS_HOST_PORT}|${REDIS_PORT}|缓存与会话
etcd client|${ETCD_HOST_PORT}|${ETCD_PORT}|服务注册
etcd peer|${ETCD_PEER_HOST_PORT}|${ETCD_PEER_PORT}|集群通信
Kafka|${KAFKA_HOST_PORT}|${KAFKA_HOST_PORT}|异步消息
Canal|${CANAL_PORT}|11111|Binlog 订阅
XXL-JOB Admin|${XXL_JOB_PORT}|8080|调度控制台
EOF

  if [ "${ENABLE_TRACE_PIPELINE_VALUE}" = "1" ]; then
    cat <<EOF
OTEL Collector gRPC|${OTEL_COLLECTOR_GRPC_HOST_PORT}|4317|OTLP gRPC 上报
OTEL Collector HTTP|${OTEL_COLLECTOR_HTTP_HOST_PORT}|4318|OTLP HTTP 上报
Jaeger UI|${JAEGER_HOST_PORT}|16686|Trace 查询界面
Jaeger Collector HTTP|14268|14268|Trace 上报入口
Jaeger Collector gRPC|14250|14250|Trace 上报入口
EOF
  fi

  if [ "${ENABLE_GRAFANA_VALUE}" = "1" ]; then
    cat <<EOF
Grafana|${GRAFANA_HOST_PORT}|3000|监控看板
EOF
  fi
}

fct_print_port_table() {
  printf '端口清单：\n'
  printf '  %-24s %-12s %-12s %s\n' "组件" "宿主机端口" "容器端口" "说明"
  while IFS='|' read -r name host_port container_port desc; do
    [ -n "${name}" ] || continue
    printf '  %-24s %-12s %-12s %s\n' "${name}" "${host_port}" "${container_port}" "${desc}"
  done < <(fct_expected_port_rows)
}

fct_check_host_port_conflicts() {
  if ! command -v ss >/dev/null 2>&1; then
    printf '未找到 ss，跳过启动前端口预检查。\n' >&2
    return 0
  fi

  local ss_lines
  ss_lines="$(ss -ltnpH || true)"

  declare -A owned_ports=()
  while IFS= read -r port; do
    [ -n "${port}" ] || continue
    owned_ports["${port}"]=1
  done < <(fct_compose_owned_ports)

  declare -a conflicts=()
  while IFS='|' read -r name host_port _ desc; do
    [ -n "${name}" ] || continue
    if [ -n "${owned_ports[${host_port}]:-}" ]; then
      continue
    fi

    if printf '%s\n' "${ss_lines}" | awk -v port=":${host_port}" '$4 ~ port"$" {found=1} END {exit !found}'; then
      local holder
      holder="$(printf '%s\n' "${ss_lines}" | awk -v port=":${host_port}" '$4 ~ port"$" {print $4 " " $NF; exit}')"
      conflicts+=("${name}|${host_port}|${desc}|${holder}")
    fi
  done < <(fct_expected_port_rows)

  if [ "${#conflicts[@]}" -eq 0 ]; then
    return 0
  fi

  printf '检测到宿主机端口冲突，本次启动已取消：\n' >&2
  for conflict in "${conflicts[@]}"; do
    IFS='|' read -r name host_port desc holder <<<"${conflict}"
    printf '  - %s 需要端口 %s：%s；当前占用：%s\n' "${name}" "${host_port}" "${desc}" "${holder:-未知占用方}" >&2
  done
  printf '请先执行 bash ./scripts/bench/run.sh ports 查看完整端口清单，或覆盖对应环境变量后重试。\n' >&2
  return 1
}

if ! command -v docker >/dev/null 2>&1; then
  echo "未找到 docker 命令" >&2
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose 不可用" >&2
  exit 1
fi

if [ ! -f "${COMPOSE_ENV_PATH}" ]; then
  echo "缺少 Compose 环境文件：${COMPOSE_ENV_PATH}" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "${COMPOSE_ENV_PATH}"
set +a

if [ -n "${ENV_OVERRIDE_ENABLE_LOG_PIPELINE_SET}" ]; then
  export ENABLE_LOG_PIPELINE="${ENV_OVERRIDE_ENABLE_LOG_PIPELINE}"
fi
if [ -n "${ENV_OVERRIDE_ENABLE_TRACE_PIPELINE_SET}" ]; then
  export ENABLE_TRACE_PIPELINE="${ENV_OVERRIDE_ENABLE_TRACE_PIPELINE}"
fi
if [ -n "${ENV_OVERRIDE_ENABLE_GRAFANA_SET}" ]; then
  export ENABLE_GRAFANA="${ENV_OVERRIDE_ENABLE_GRAFANA}"
fi
if [ -n "${ENV_OVERRIDE_XXL_JOB_PORT_SET}" ]; then
  export XXL_JOB_PORT="${ENV_OVERRIDE_XXL_JOB_PORT}"
fi

cd "${DEPLOY_DIR}"
export RESEED_MYSQL RESET_CANAL

readonly ENABLE_LOG_PIPELINE_VALUE="${ENABLE_LOG_PIPELINE:-0}"
readonly ENABLE_TRACE_PIPELINE_VALUE="${ENABLE_TRACE_PIPELINE:-0}"
readonly ENABLE_GRAFANA_VALUE="${ENABLE_GRAFANA:-0}"

app_services=(front-api user-rpc content-rpc interaction-rpc count-rpc search-rpc search-indexer)
app_images=(
  "${FRONT_API_IMAGE}"
  "${USER_RPC_IMAGE}"
  "${CONTENT_RPC_IMAGE}"
  "${INTERACTION_RPC_IMAGE}"
  "${COUNT_RPC_IMAGE}"
  "${SEARCH_RPC_IMAGE}"
  "${SEARCH_INDEXER_IMAGE}"
)
services=(
  etcd
  redis
  mysql
  kafka
  canal
  opensearch
  search-index-bootstrap
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
if [ "${ENABLE_GRAFANA_VALUE}" = "1" ]; then
  services+=(grafana)
fi

fct_check_host_port_conflicts

missing_services=()
for index in "${!app_services[@]}"; do
  if ! docker image inspect "${app_images[${index}]}" >/dev/null 2>&1; then
    missing_services+=("${app_services[${index}]}")
  fi
done

if [ "${REBUILD}" = "1" ]; then
  echo "开始构建应用镜像..."
  docker compose --env-file .env -f docker-compose.yml build "${app_services[@]}"
elif [ "${#missing_services[@]}" -gt 0 ]; then
  echo "开始补齐缺失的应用镜像：${missing_services[*]}"
  docker compose --env-file .env -f docker-compose.yml build "${missing_services[@]}"
fi

echo "开始启动 zfeed Docker 栈..."
docker compose --env-file .env -f docker-compose.yml up -d --remove-orphans "${services[@]}"

printf 'zfeed Docker 栈已进入启动流程。\n'
printf '  网关入口： http://127.0.0.1:%s\n' "${GATEWAY_HOST_PORT:-18080}"
printf '  API 直连： http://127.0.0.1:%s\n' "${FRONT_API_PORT:-5000}"
printf '  停止命令： bash ./scripts/stop.sh\n'
printf '  全量重建： REBUILD=1 RESEED_MYSQL=1 RESET_CANAL=1 bash ./scripts/start.sh\n'
fct_print_port_table
