#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR
readonly GO_BIN="${GO_BIN:-go}"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/report.sh <result-dir> [--baseline <baseline-result-dir>]

说明：
  从 bench/results 下的一次压测结果生成 report.md。
EOF
}

fct_main() {
	local result_dir="${1:-}"
	if [[ -z "${result_dir}" || "${result_dir}" == "-h" || "${result_dir}" == "--help" ]]; then
		fct_usage
		return 0
	fi
	shift || true

	local -a args=()
	while [[ "$#" -gt 0 ]]; do
		case "${1}" in
		--baseline)
			if [[ "$#" -lt 2 ]]; then
				printf '--baseline 需要一个结果目录参数。\n' >&2
				return 2
			fi
			args+=("--baseline" "${2}")
			shift 2
			;;
		*)
			printf '未知参数：%s\n' "${1}" >&2
			fct_usage >&2
			return 2
			;;
		esac
	done

	"${GO_BIN}" run ./bench/tools/benchreport "${args[@]}" "${result_dir}"
}

cd "${ROOT_DIR}"
fct_main "$@"
