#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR

readonly GO_BIN="${GO_BIN:-go}"
readonly RESULTS_ROOT="${RESULTS_ROOT:-bench/results}"
readonly BENCH_COUNT="${BENCH_COUNT:-3}"
readonly BENCH_TIME_THRESHOLD="${BENCH_TIME_THRESHOLD:-0.10}"
readonly BENCH_BYTES_THRESHOLD="${BENCH_BYTES_THRESHOLD:-0.10}"
readonly BENCH_ALLOCS_THRESHOLD="${BENCH_ALLOCS_THRESHOLD:-0.10}"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/ci-gate.sh

环境变量：
  BENCH_BASELINE          可选，baseline go-bench.txt 或包含 go-bench.txt 的结果目录
  BENCH_BASELINE_REF      可选，git ref；脚本会在临时 worktree 中生成 baseline
  BENCH_COUNT             Go benchmark 重复次数，默认 3
  RESULTS_ROOT            结果目录，默认 bench/results
  BENCH_TIME_THRESHOLD    ns/op 允许回退比例，默认 0.10
  BENCH_BYTES_THRESHOLD   B/op 允许回退比例，默认 0.10
  BENCH_ALLOCS_THRESHOLD  allocs/op 允许回退比例，默认 0.10
EOF
}

fct_result_root() {
	case "${RESULTS_ROOT}" in
	/*)
		printf '%s\n' "${RESULTS_ROOT}"
		;;
	*)
		printf '%s/%s\n' "${ROOT_DIR}" "${RESULTS_ROOT}"
		;;
	esac
}

fct_latest_go_bench_result() {
	find "$(fct_result_root)" -maxdepth 1 -type d -name '*-go-bench' -printf '%T@ %p\n' 2>/dev/null |
		sort -nr |
		awk 'NR == 1 { print $2 }'
}

fct_resolve_go_bench_file() {
	local input="${1}"
	if [[ -f "${input}" ]]; then
		printf '%s\n' "${input}"
		return 0
	fi
	if [[ -f "${input}/go-bench.txt" ]]; then
		printf '%s/go-bench.txt\n' "${input}"
		return 0
	fi
	printf '无法找到 go-bench.txt：%s\n' "${input}" >&2
	return 2
}

fct_generate_baseline_from_ref() {
	local ref="${1}"
	local worktree_dir
	worktree_dir="$(mktemp -d)"
	rm -rf "${worktree_dir}"

	if ! git -C "${ROOT_DIR}" rev-parse --verify "${ref}^{commit}" >/dev/null 2>&1; then
		printf 'baseline ref 不存在或不是 commit：%s\n' "${ref}" >&2
		return 1
	fi

	if ! git -C "${ROOT_DIR}" worktree add --detach "${worktree_dir}" "${ref}" >/dev/null; then
		rm -rf "${worktree_dir}"
		printf 'baseline ref 创建 worktree 失败：%s\n' "${ref}" >&2
		return 1
	fi

	local baseline_results="${worktree_dir}/.bench-baseline-results"
	if ! env BENCH_COUNT="${BENCH_COUNT}" RESULTS_ROOT="${baseline_results}" bash "${worktree_dir}/scripts/bench/run.sh" go-bench >/dev/null; then
		git -C "${ROOT_DIR}" worktree remove --force "${worktree_dir}" >/dev/null 2>&1 || true
		printf 'baseline ref 运行 go-bench 失败：%s\n' "${ref}" >&2
		return 1
	fi
	local baseline_file
	baseline_file="$(find "${baseline_results}" -maxdepth 2 -type f -name go-bench.txt -print -quit)"
	if [[ -z "${baseline_file}" ]]; then
		git -C "${ROOT_DIR}" worktree remove --force "${worktree_dir}" >/dev/null 2>&1 || true
		printf 'baseline ref 未生成 go-bench.txt：%s\n' "${ref}" >&2
		return 1
	fi

	local stable_file
	stable_file="$(mktemp)"
	cp "${baseline_file}" "${stable_file}"
	git -C "${ROOT_DIR}" worktree remove --force "${worktree_dir}" >/dev/null 2>&1 || true
	printf '%s\n' "${stable_file}"
}

fct_cleanup_generated_baseline() {
	local generated="${1}"
	local baseline_file="${2}"
	if [[ "${generated}" == "1" && -n "${baseline_file}" ]]; then
		rm -f "${baseline_file}"
	fi
}

fct_main() {
	if [[ "${1:-}" == "-h" || "${1:-}" == "--help" || "${1:-}" == "help" ]]; then
		fct_usage
		return 0
	fi

	local baseline_file=""
	local generated_baseline=0
	local baseline_input="${BENCH_BASELINE:-}"
	if [[ -n "${BENCH_BASELINE_REF:-}" ]]; then
		baseline_input="$(fct_generate_baseline_from_ref "${BENCH_BASELINE_REF}")"
		generated_baseline=1
	fi
	if [[ -n "${baseline_input}" ]]; then
		if ! baseline_file="$(fct_resolve_go_bench_file "${baseline_input}")"; then
			fct_cleanup_generated_baseline "${generated_baseline}" "${baseline_file}"
			return 1
		fi
	fi

	if ! env BENCH_COUNT="${BENCH_COUNT}" RESULTS_ROOT="${RESULTS_ROOT}" bash "${ROOT_DIR}/scripts/bench/run.sh" go-bench; then
		fct_cleanup_generated_baseline "${generated_baseline}" "${baseline_file}"
		return 1
	fi

	local current_result
	current_result="$(fct_latest_go_bench_result)"
	if [[ -z "${current_result}" ]]; then
		printf '未找到当前 go-bench 结果目录。\n' >&2
		fct_cleanup_generated_baseline "${generated_baseline}" "${baseline_file}"
		return 1
	fi

	local current_file="${current_result}/go-bench.txt"
	if [[ -z "${baseline_file}" ]]; then
		fct_cleanup_generated_baseline "${generated_baseline}" "${baseline_file}"
		printf '未提供 BENCH_BASELINE，仅完成当前 benchmark：%s\n' "${current_file}"
		return 0
	fi

	local gate_status=0
	"${GO_BIN}" run ./bench/tools/benchgate \
		--baseline "${baseline_file}" \
		--current "${current_file}" \
		--time-threshold "${BENCH_TIME_THRESHOLD}" \
		--bytes-threshold "${BENCH_BYTES_THRESHOLD}" \
		--allocs-threshold "${BENCH_ALLOCS_THRESHOLD}" || gate_status=$?
	fct_cleanup_generated_baseline "${generated_baseline}" "${baseline_file}"
	return "${gate_status}"
}

fct_main "$@"
