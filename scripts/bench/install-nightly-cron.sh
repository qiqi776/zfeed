#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR

readonly BENCH_NIGHTLY_CRON_SCHEDULE="${BENCH_NIGHTLY_CRON_SCHEDULE:-17 3 * * *}"
readonly BENCH_NIGHTLY_CRON_LOG="${BENCH_NIGHTLY_CRON_LOG:-${ROOT_DIR}/bench/results/nightly/nightly-gate.log}"
readonly BENCH_NIGHTLY_CRON_MARKER="# zfeed benchmark nightly gate"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/install-nightly-cron.sh [--dry-run|--apply]

说明：
  默认 --dry-run，只打印将要写入 crontab 的内容。
  --apply 会更新当前用户 crontab，并替换已有的 zfeed benchmark nightly gate 条目。

环境变量：
  BENCH_NIGHTLY_CRON_SCHEDULE  cron 表达式，默认 17 3 * * *
  BENCH_NIGHTLY_CRON_LOG       日志路径，默认 bench/results/nightly/nightly-gate.log
EOF
}

fct_cron_line() {
	printf '%s cd %q && bash scripts/bench/nightly-gate.sh run >>%q 2>&1 %s\n' \
		"${BENCH_NIGHTLY_CRON_SCHEDULE}" \
		"${ROOT_DIR}" \
		"${BENCH_NIGHTLY_CRON_LOG}" \
		"${BENCH_NIGHTLY_CRON_MARKER}"
}

fct_install() {
	local mode="${1:---dry-run}"
	case "${mode}" in
	-h | --help | help)
		fct_usage
		;;
	--dry-run | dry-run)
		printf 'dry-run: would install crontab line:\n'
		fct_cron_line
		;;
	--apply | apply)
		local tmp_file
		tmp_file="$(mktemp)"
		trap 'rm -f "${tmp_file}"' EXIT
		crontab -l 2>/dev/null | grep -vF "${BENCH_NIGHTLY_CRON_MARKER}" >"${tmp_file}" || true
		fct_cron_line >>"${tmp_file}"
		crontab "${tmp_file}"
		printf 'nightly cron 已安装。\n'
		;;
	*)
		printf '未知模式：%s\n' "${mode}" >&2
		fct_usage >&2
		return 2
		;;
	esac
}

fct_install "${1:---dry-run}"
