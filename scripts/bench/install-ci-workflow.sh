#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR

readonly TEMPLATE="${ROOT_DIR}/bench/ci/github-actions-benchmark.yml"
readonly TARGET="${ROOT_DIR}/.github/workflows/zfeed-benchmark-gate.yml"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/install-ci-workflow.sh [--dry-run|--apply]

说明：
  默认 --dry-run，只打印将要执行的复制动作。
  --apply 会创建 .github/workflows/zfeed-benchmark-gate.yml。
EOF
}

fct_install() {
	local mode="${1:---dry-run}"
	case "${mode}" in
	-h | --help | help)
		fct_usage
		;;
	--dry-run | dry-run)
		printf 'dry-run: mkdir -p %q\n' "$(dirname "${TARGET}")"
		printf 'dry-run: cp %q %q\n' "${TEMPLATE}" "${TARGET}"
		;;
	--apply | apply)
		mkdir -p "$(dirname "${TARGET}")"
		cp "${TEMPLATE}" "${TARGET}"
		printf 'CI workflow 已安装：%s\n' "${TARGET}"
		;;
	*)
		printf '未知模式：%s\n' "${mode}" >&2
		fct_usage >&2
		return 2
		;;
	esac
}

fct_install "${1:---dry-run}"
