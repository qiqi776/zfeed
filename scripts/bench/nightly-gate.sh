#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR

readonly BENCH_RUN_SCRIPT="${BENCH_RUN_SCRIPT:-${ROOT_DIR}/scripts/bench/run.sh}"
readonly BENCH_CI_GATE_SCRIPT="${BENCH_CI_GATE_SCRIPT:-${ROOT_DIR}/scripts/bench/ci-gate.sh}"
readonly RESULTS_ROOT="${RESULTS_ROOT:-bench/results}"
readonly BENCH_NIGHTLY_STATE_DIR="${BENCH_NIGHTLY_STATE_DIR:-bench/results/nightly}"
readonly BENCH_NIGHTLY_COUNT="${BENCH_NIGHTLY_COUNT:-${BENCH_COUNT:-10}}"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/nightly-gate.sh <命令>

命令：
  seed      运行一次 go-bench，并保存为 nightly baseline
  run       使用已保存 baseline 执行 nightly benchmark 回退门禁
  status    打印 nightly gate 最近一次状态

环境变量：
  BENCH_NIGHTLY_STATE_DIR  nightly 状态目录，默认 bench/results/nightly
  RESULTS_ROOT             benchmark 结果目录，默认 bench/results
  BENCH_RUN_SCRIPT         run.sh 路径，默认 scripts/bench/run.sh
  BENCH_CI_GATE_SCRIPT     ci-gate.sh 路径，默认 scripts/bench/ci-gate.sh
  BENCH_NIGHTLY_COUNT      Go benchmark 重复次数，默认 10
  BENCH_COUNT              Go benchmark 重复次数，未设置 BENCH_NIGHTLY_COUNT 时作为兼容输入

示例：
  scripts/bench/nightly-gate.sh seed
  scripts/bench/nightly-gate.sh run
EOF
}

fct_abs_path() {
	local path="${1}"
	case "${path}" in
	/*)
		printf '%s\n' "${path}"
		;;
	*)
		printf '%s/%s\n' "${ROOT_DIR}" "${path}"
		;;
	esac
}

fct_results_root() {
	fct_abs_path "${RESULTS_ROOT}"
}

fct_state_dir() {
	fct_abs_path "${BENCH_NIGHTLY_STATE_DIR}"
}

fct_baseline_file() {
	printf '%s/baseline-go-bench.txt\n' "$(fct_state_dir)"
}

fct_last_run_file() {
	printf '%s/last-run.env\n' "$(fct_state_dir)"
}

fct_latest_go_bench_result() {
	find "$(fct_results_root)" -maxdepth 1 -type d -name '*-go-bench' -printf '%T@ %p\n' 2>/dev/null |
		sort -nr |
		awk 'NR == 1 { print $2 }'
}

fct_latest_go_bench_file() {
	local result_dir
	result_dir="$(fct_latest_go_bench_result)"
	if [[ -z "${result_dir}" || ! -f "${result_dir}/go-bench.txt" ]]; then
		printf '未找到最新 go-bench.txt，请先运行 go-bench。\n' >&2
		return 1
	fi
	printf '%s/go-bench.txt\n' "${result_dir}"
}

fct_write_last_run() {
	local status="${1}"
	local baseline="${2}"
	local current="${3}"
	local gate_status="${4}"
	local state_dir
	state_dir="$(fct_state_dir)"
	mkdir -p "${state_dir}"
	cat >"$(fct_last_run_file)" <<EOF
status=${status}
baseline=${baseline}
current=${current}
gate_status=${gate_status}
updated_at=$(date -Is)
EOF
}

fct_seed() {
	mkdir -p "$(fct_results_root)" "$(fct_state_dir)"
	env BENCH_COUNT="${BENCH_NIGHTLY_COUNT}" RESULTS_ROOT="$(fct_results_root)" bash "${BENCH_RUN_SCRIPT}" go-bench

	local current_file
	current_file="$(fct_latest_go_bench_file)"
	cp "${current_file}" "$(fct_baseline_file)"
	fct_write_last_run "SEEDED" "$(fct_baseline_file)" "${current_file}" "0"
	printf 'nightly baseline 已更新：%s\n' "$(fct_baseline_file)"
}

fct_run() {
	mkdir -p "$(fct_results_root)" "$(fct_state_dir)"

	local baseline_file
	baseline_file="$(fct_baseline_file)"
	if [[ ! -f "${baseline_file}" ]]; then
		printf '未找到 nightly baseline：%s\n' "${baseline_file}" >&2
		printf '请先执行：scripts/bench/nightly-gate.sh seed\n' >&2
		return 2
	fi

	local gate_status=0
	env BENCH_BASELINE="${baseline_file}" BENCH_COUNT="${BENCH_NIGHTLY_COUNT}" RESULTS_ROOT="$(fct_results_root)" bash "${BENCH_CI_GATE_SCRIPT}" || gate_status=$?
	if [[ "${gate_status}" -ne 0 ]]; then
		fct_write_last_run "FAILED" "${baseline_file}" "-" "${gate_status}"
		return "${gate_status}"
	fi

	local current_file
	current_file="$(fct_latest_go_bench_file)"
	cp "${current_file}" "${baseline_file}"
	fct_write_last_run "PASSED" "${baseline_file}" "${current_file}" "0"
	printf 'nightly benchmark gate passed，baseline 已轮转：%s\n' "${baseline_file}"
}

fct_status() {
	local status_file
	status_file="$(fct_last_run_file)"
	if [[ ! -f "${status_file}" ]]; then
		printf '未找到 nightly gate 状态：%s\n' "${status_file}" >&2
		return 2
	fi
	cat "${status_file}"
}

fct_main() {
	local command="${1:-}"
	case "${command}" in
	-h | --help | help | "")
		fct_usage
		;;
	seed)
		fct_seed
		;;
	run)
		fct_run
		;;
	status)
		fct_status
		;;
	*)
		printf '未知命令：%s\n' "${command}" >&2
		fct_usage >&2
		return 2
		;;
	esac
}

fct_main "$@"
