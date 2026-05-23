#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR
readonly COMPOSE_ENV_PATH="${ROOT_DIR}/deploy/.env"

readonly K6_BIN="${K6_BIN:-k6}"
readonly GHZ_BIN="${GHZ_BIN:-ghz}"
readonly GO_BIN="${GO_BIN:-go}"
readonly BASE_URL="${BASE_URL:-http://127.0.0.1:18080}"
readonly DATA_DIR="${DATA_DIR:-bench/data/small}"
readonly RESULTS_ROOT="${RESULTS_ROOT:-bench/results}"
readonly BENCH_COUNT="${BENCH_COUNT:-10}"
readonly GHZ_CONTENT_TARGET="${GHZ_CONTENT_TARGET:-127.0.0.1:5001}"
readonly GHZ_INTERACTION_TARGET="${GHZ_INTERACTION_TARGET:-127.0.0.1:5002}"
readonly GHZ_SEARCH_TARGET="${GHZ_SEARCH_TARGET:-127.0.0.1:5006}"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/run.sh <命令>

命令：
  start-stack    启动本地压测与观测栈
  smoke          运行 k6 冒烟场景
  load           运行 k6 混合负载场景
  stress         运行 k6 写重压场景
  spike          运行 k6 脉冲流量场景
  soak           运行 k6 长稳压场景
  search         运行 k6 搜索场景
  hot-content    运行 k6 热点内容场景
  go-bench       运行 feed/search/count/hotrank Go benchmark
  ghz            运行 bench/ghz 中的 gRPC 定点压测
  pprof          从 PPROF_URL 抓取 pprof top 输出
  ports          打印项目默认端口清单

环境变量：
  BASE_URL       HTTP 压测目标，默认 http://127.0.0.1:18080
  DATA_DIR       k6 数据目录，默认 bench/data/small
  RESULTS_ROOT   结果归档根目录，默认 bench/results
  K6_BIN         k6 可执行文件，默认 k6
  GHZ_BIN        ghz 可执行文件，默认 ghz
  GHZ_CONTENT_TARGET      content-rpc ghz 目标，默认 127.0.0.1:5001
  GHZ_INTERACTION_TARGET  interaction-rpc ghz 目标，默认 127.0.0.1:5002
  GHZ_SEARCH_TARGET       search-rpc ghz 目标，默认 127.0.0.1:5006
  BENCH_COUNT    Go benchmark 重复次数，默认 10
  PPROF_URL      pprof 地址，例如 http://127.0.0.1:6060/debug/pprof/profile?seconds=30
EOF
}

fct_require_bin() {
	local bin_name="${1}"
	if ! command -v "${bin_name}" >/dev/null 2>&1; then
		printf '缺少必需命令：%s\n' "${bin_name}" >&2
		return 1
	fi
}

fct_commit() {
	git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || printf 'nogit'
}

fct_result_dir() {
	local scenario="${1}"
	local ts
	ts="$(date '+%Y%m%d-%H%M%S')"
	printf '%s/%s-%s-%s\n' "$(fct_results_root)" "${ts}" "$(fct_commit)" "${scenario}"
}

fct_results_root() {
	case "${RESULTS_ROOT}" in
	/*)
		printf '%s\n' "${RESULTS_ROOT}"
		;;
	*)
		printf '%s/%s\n' "${ROOT_DIR}" "${RESULTS_ROOT}"
		;;
	esac
}

fct_data_dir() {
	case "${DATA_DIR}" in
	/*)
		printf '%s\n' "${DATA_DIR}"
		;;
	*)
		printf '%s/%s\n' "${ROOT_DIR}" "${DATA_DIR}"
		;;
	esac
}

fct_prepare_result_dir() {
	local result_dir="${1}"
	local scenario="${2}"
	mkdir -p "${result_dir}"
	fct_write_env_report "${result_dir}" "${scenario}"
	fct_write_result_readme "${result_dir}" "${scenario}"
}

fct_write_env_report() {
	local result_dir="${1}"
	local scenario="${2}"
	cat >"${result_dir}/env.md" <<EOF
# 压测环境

- 场景：${scenario}
- 提交：$(fct_commit)
- 基础 URL：${BASE_URL}
- 数据目录：$(fct_data_dir)
- 开始时间：$(date -Is)
- K6 可执行文件：${K6_BIN}
- GHZ 可执行文件：${GHZ_BIN}
- GHZ 内容目标：${GHZ_CONTENT_TARGET}
- GHZ 互动目标：${GHZ_INTERACTION_TARGET}
- GHZ 搜索目标：${GHZ_SEARCH_TARGET}
- Go 可执行文件：${GO_BIN}
- 压测次数：${BENCH_COUNT}
EOF
}

fct_write_result_readme() {
	local result_dir="${1}"
	local scenario="${2}"
	cat >"${result_dir}/README.md" <<EOF
# 压测结果说明

- 场景：${scenario}
- 结果目录：$(basename "${result_dir}")
- 生成时间：$(date -Is)

本目录中的文件以"原始结果 + 中文说明"的方式组织：

- \`README.md\`：当前目录说明，帮助快速判断每个文件的用途。
- \`env.md\`：记录本次压测环境、目标地址、二进制路径和关键参数。
- \`k6-summary.json\`：k6 结构化汇总，仅 HTTP 场景生成，可供后续脚本二次分析。
- \`k6-output.txt\`：k6 原始终端输出，包含检查项、时延、吞吐和错误率。
- \`go-bench.txt\`：Go benchmark 原始输出，仅 \`go-bench\` 场景生成。
- \`*.txt\`：ghz 或 pprof 原始文本输出；ghz 按场景名生成，pprof 使用 \`pprof-top.txt\`。

如果某个文件不存在，通常表示当前场景不会产出该类型结果；新的脚本也会在真正发起请求前先做目标可达性检查，避免"服务未启动但已生成一份看起来像结果的目录"。
EOF
}

fct_check_http_target() {
	local target_url="${1}"
	fct_require_bin curl
	if ! curl -sS -o /dev/null --connect-timeout 2 --max-time 3 "${target_url}"; then
		printf 'HTTP 压测目标不可达：%s\n' "${target_url}" >&2
		printf '请先执行 bash ./scripts/bench/run.sh start-stack 或 bash ./scripts/start.sh，待服务启动后再重试。\n' >&2
		return 1
	fi
}

fct_check_tcp_target() {
	local target="${1}"
	local label="${2}"
	local host="${target%:*}"
	local port="${target##*:}"

	if [[ -z "${host}" || -z "${port}" || "${host}" == "${target}" ]]; then
		printf '无效的 gRPC 目标地址：%s\n' "${target}" >&2
		return 2
	fi

	if command -v timeout >/dev/null 2>&1; then
		if ! timeout 3 bash -c ": >/dev/tcp/${host}/${port}" >/dev/null 2>&1; then
			printf '%s 不可达：%s\n' "${label}" "${target}" >&2
			printf '请先确认目标服务已经监听对应端口，再重新执行 ghz 压测。\n' >&2
			return 1
		fi
		return 0
	fi

	if ! (: >"/dev/tcp/${host}/${port}") >/dev/null 2>&1; then
		printf '%s 不可达：%s\n' "${label}" "${target}" >&2
		printf '请先确认目标服务已经监听对应端口，再重新执行 ghz 压测。\n' >&2
		return 1
	fi
}

fct_check_pprof_target() {
	local pprof_url="${1}"
	fct_require_bin curl
	if ! curl -fsS -o /dev/null --connect-timeout 2 --max-time 5 "${pprof_url}"; then
		printf 'pprof 地址不可达：%s\n' "${pprof_url}" >&2
		printf '请先确认目标服务已开启 pprof，再重新抓取。\n' >&2
		return 1
	fi
}

fct_print_ports() {
	if [[ ! -f "${COMPOSE_ENV_PATH}" ]]; then
		printf '未找到端口配置文件：%s\n' "${COMPOSE_ENV_PATH}" >&2
		return 1
	fi

	(
		set -a
		# shellcheck disable=SC1090
		. "${COMPOSE_ENV_PATH}"
		set +a

		cat <<EOF
# zfeed 默认端口清单

## 核心服务

| 组件 | 宿主机端口 | 容器端口 | 说明 |
| --- | ---: | ---: | --- |
| nginx 网关 | ${GATEWAY_HOST_PORT} | 80 | HTTP 压测默认入口 |
| front-api | ${FRONT_API_PORT} | 5000 | HTTP 直连入口 |
| front-api Prometheus | ${PROM_PORT} | 9290 | API 指标 |
| content-rpc | 5001 | 5001 | 内容服务 gRPC |
| content-rpc Prometheus | ${CONTENT_PROM_PORT} | 9291 | 内容服务指标 |
| content-rpc XXL 执行器 | ${XXL_JOB_EXECUTOR_PORT} | ${XXL_EXECUTOR_PORT} | XXL-JOB 回调执行器 |
| interaction-rpc | 5002 | 5002 | 互动服务 gRPC |
| interaction-rpc Prometheus | ${INTERACTION_PROM_PORT} | 9293 | 互动服务指标 |
| user-rpc | 5003 | 5003 | 用户服务 gRPC |
| user-rpc Prometheus | ${USER_PROM_PORT} | 9294 | 用户服务指标 |
| count-rpc | 5004 | 5004 | 计数服务 gRPC |
| count-rpc Prometheus | ${COUNT_PROM_PORT} | 9292 | 计数服务指标 |
| search-rpc | 5006 | 5006 | 搜索服务 gRPC |
| search-rpc Prometheus | ${SEARCH_PROM_PORT} | 9295 | 搜索服务指标 |
| MySQL | ${MYSQL_PORT} | 3306 | 主存储 |
| Redis | ${REDIS_HOST_PORT} | ${REDIS_PORT} | 缓存与会话 |
| etcd client | ${ETCD_HOST_PORT} | ${ETCD_PORT} | 服务注册 |
| etcd peer | ${ETCD_PEER_HOST_PORT} | ${ETCD_PEER_PORT} | etcd 节点通信 |
| Kafka | ${KAFKA_HOST_PORT} | ${KAFKA_HOST_PORT} | 异步消息 |
| Canal | ${CANAL_PORT} | 11111 | Binlog 订阅 |
| XXL-JOB Admin | ${XXL_JOB_PORT} | 8080 | 调度控制台 |
| Prometheus | ${PROMETHEUS_HOST_PORT} | 9090 | 指标查询 |

## 可选观测组件

| 组件 | 宿主机端口 | 容器端口 | 启用条件 |
| --- | ---: | ---: | --- |
| Grafana | ${GRAFANA_HOST_PORT} | 3000 | ENABLE_GRAFANA=1 |
| OTEL Collector gRPC | ${OTEL_COLLECTOR_GRPC_HOST_PORT} | 4317 | ENABLE_TRACE_PIPELINE=1 |
| OTEL Collector HTTP | ${OTEL_COLLECTOR_HTTP_HOST_PORT} | 4318 | ENABLE_TRACE_PIPELINE=1 |
| Jaeger UI | ${JAEGER_HOST_PORT} | 16686 | ENABLE_TRACE_PIPELINE=1 |
| Jaeger Collector HTTP | 14268 | 14268 | ENABLE_TRACE_PIPELINE=1 |
| Jaeger Collector gRPC | 14250 | 14250 | ENABLE_TRACE_PIPELINE=1 |

说明：

- 当前默认已将 \`XXL_JOB_PORT\` 设为 ${XXL_JOB_PORT}，避免与常见本地开发端口 8081 冲突。
- 如果需要临时避让端口，可在启动前覆盖环境变量，例如：\`XXL_JOB_PORT=28081 bash ./scripts/bench/run.sh start-stack\`。
EOF
	)
}

fct_run_k6() {
	local scenario="${1}"
	local scenario_file="${ROOT_DIR}/bench/k6/scenarios/${scenario}.js"
	if [[ ! -f "${scenario_file}" ]]; then
		printf '未知的 k6 场景：%s\n' "${scenario}" >&2
		return 2
	fi
	fct_require_bin "${K6_BIN}"
	fct_check_http_target "${BASE_URL}"

	local result_dir
	result_dir="$(fct_result_dir "${scenario}")"
	fct_prepare_result_dir "${result_dir}" "${scenario}"

	printf '开始执行 k6 场景：%s\n' "${scenario}" >&2
	printf '结果目录：%s\n' "${result_dir}" >&2
	env BASE_URL="${BASE_URL}" DATA_DIR="$(fct_data_dir)" "${K6_BIN}" run \
		--summary-export "${result_dir}/k6-summary.json" \
		"${scenario_file}" 2>&1 | tee "${result_dir}/k6-output.txt"
}

fct_run_go_bench() {
	fct_require_bin "${GO_BIN}"

	local result_dir
	result_dir="$(fct_result_dir "go-bench")"
	fct_prepare_result_dir "${result_dir}" "go-bench"

	printf '开始执行 Go benchmark\n' >&2
	printf '结果目录：%s\n' "${result_dir}" >&2
	GOCACHE="${GOCACHE:-/tmp/go-build}" "${GO_BIN}" test \
		-run '^$' \
		-bench=. \
		-benchmem \
		-count="${BENCH_COUNT}" \
		./app/rpc/content/internal/logic/feed \
		./app/rpc/search/search-rpc/internal/querynorm \
		./app/rpc/count/internal/logic \
		./pkg/hotrank 2>&1 | tee "${result_dir}/go-bench.txt"
}

fct_run_ghz() {
	fct_require_bin "${GHZ_BIN}"

	local config
	local found_config=0
	for config in "${ROOT_DIR}"/bench/ghz/*.json; do
		[[ -f "${config}" ]] || continue
		found_config=1
		local name
		name="$(basename "${config}" .json)"
		local target
		target="$(fct_ghz_target_for "${name}")"
		fct_check_tcp_target "${target}" "gRPC 压测目标 ${name}"
	done

	if [[ "${found_config}" -eq 0 ]]; then
		printf '未找到任何 ghz 配置文件：%s\n' "${ROOT_DIR}/bench/ghz/*.json" >&2
		return 2
	fi

	local result_dir
	result_dir="$(fct_result_dir "ghz")"
	fct_prepare_result_dir "${result_dir}" "ghz"

	for config in "${ROOT_DIR}"/bench/ghz/*.json; do
		[[ -f "${config}" ]] || continue
		local name
		name="$(basename "${config}" .json)"
		local target
		target="$(fct_ghz_target_for "${name}")"
		printf '开始执行 ghz 场景：%s\n' "${name}" >&2
		printf '目标地址：%s\n' "${target}" >&2
		"${GHZ_BIN}" --config "${config}" "${target}" 2>&1 | tee "${result_dir}/${name}.txt"
	done
}

fct_run_pprof() {
	fct_require_bin "${GO_BIN}"
	local pprof_url="${PPROF_URL:-}"
	if [[ -z "${pprof_url}" ]]; then
		printf '抓取 pprof 前必须提供 PPROF_URL。\n' >&2
		return 2
	fi
	fct_check_pprof_target "${pprof_url}"

	local result_dir
	result_dir="$(fct_result_dir "pprof")"
	fct_prepare_result_dir "${result_dir}" "pprof"

	printf '开始抓取 pprof：%s\n' "${pprof_url}" >&2
	printf '结果目录：%s\n' "${result_dir}" >&2
	"${GO_BIN}" tool pprof -top "${pprof_url}" 2>&1 | tee "${result_dir}/pprof-top.txt"
}

fct_ghz_target_for() {
	local name="${1}"
	case "${name}" in
	feed-*)
		printf '%s\n' "${GHZ_CONTENT_TARGET}"
		;;
	like-*)
		printf '%s\n' "${GHZ_INTERACTION_TARGET}"
		;;
	search-*)
		printf '%s\n' "${GHZ_SEARCH_TARGET}"
		;;
	*)
		printf '%s\n' "${GHZ_CONTENT_TARGET}"
		;;
	esac
}

fct_start_stack() {
	ENABLE_TRACE_PIPELINE="${ENABLE_TRACE_PIPELINE:-1}" \
		ENABLE_LOG_PIPELINE="${ENABLE_LOG_PIPELINE:-1}" \
		ENABLE_GRAFANA="${ENABLE_GRAFANA:-1}" \
		bash "${ROOT_DIR}/scripts/start.sh"
}

fct_main() {
	local command="${1:-}"
	case "${command}" in
	-h | --help | help | "")
		fct_usage
		;;
	start-stack)
		fct_start_stack
		;;
	smoke)
		fct_run_k6 "smoke"
		;;
	load)
		fct_run_k6 "mixed"
		;;
	stress)
		fct_run_k6 "write-heavy"
		;;
	spike | soak | search | hot-content)
		fct_run_k6 "${command}"
		;;
	go-bench)
		fct_run_go_bench
		;;
	ghz)
		fct_run_ghz
		;;
	pprof)
		fct_run_pprof
		;;
	ports)
		fct_print_ports
		;;
	*)
		printf '未知命令：%s\n' "${command}" >&2
		fct_usage >&2
		return 2
		;;
	esac
}

cd "${ROOT_DIR}"
fct_main "$@"
