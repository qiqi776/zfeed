#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
readonly ROOT_DIR

TMP_DIR=""

fct_cleanup() {
	if [[ -n "${TMP_DIR}" ]]; then
		rm -rf "${TMP_DIR}"
	fi
}
trap fct_cleanup EXIT

fct_assert_file() {
	local path="${1}"
	if [[ ! -f "${path}" ]]; then
		printf 'expected file to exist: %s\n' "${path}" >&2
		return 1
	fi
}

fct_assert_contains() {
	local path="${1}"
	local pattern="${2}"
	if ! grep -qF "${pattern}" "${path}"; then
		printf 'expected %s to contain: %s\n' "${path}" "${pattern}" >&2
		printf 'actual content:\n' >&2
		cat "${path}" >&2
		return 1
	fi
}

fct_write_stubs() {
	local bin_dir="${1}"
	mkdir -p "${bin_dir}"

	cat >"${bin_dir}/run.sh" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${1:-}" != "go-bench" ]]; then
	printf 'unexpected run command: %s\n' "${1:-}" >&2
	exit 2
fi

result_dir="${RESULTS_ROOT}/20260101-000000-stub-go-bench"
mkdir -p "${result_dir}"
if [[ -n "${STUB_RUN_ENV_FILE:-}" ]]; then
	printf 'BENCH_COUNT=%s\n' "${BENCH_COUNT:-}" >"${STUB_RUN_ENV_FILE}"
fi
printf '%s\n' "${STUB_GO_BENCH_CONTENT:-BenchmarkStub 1 100 ns/op 0 B/op 0 allocs/op}" >"${result_dir}/go-bench.txt"
EOF
	chmod +x "${bin_dir}/run.sh"

	cat >"${bin_dir}/ci-gate.sh" <<'EOF'
#!/usr/bin/env bash
set -Eeuo pipefail

printf 'BENCH_BASELINE=%s\n' "${BENCH_BASELINE:-}" >"${STUB_GATE_ENV_FILE}"
printf 'RESULTS_ROOT=%s\n' "${RESULTS_ROOT:-}" >>"${STUB_GATE_ENV_FILE}"
printf 'BENCH_COUNT=%s\n' "${BENCH_COUNT:-}" >>"${STUB_GATE_ENV_FILE}"

result_dir="${RESULTS_ROOT}/20260101-000001-stub-go-bench"
mkdir -p "${result_dir}"
printf '%s\n' "${STUB_CURRENT_GO_BENCH_CONTENT:-BenchmarkStub 1 90 ns/op 0 B/op 0 allocs/op}" >"${result_dir}/go-bench.txt"
exit "${STUB_GATE_EXIT:-0}"
EOF
	chmod +x "${bin_dir}/ci-gate.sh"
}

fct_test_run_requires_seeded_baseline() {
	TMP_DIR="$(mktemp -d)"
	local state_dir="${TMP_DIR}/state"
	local output="${TMP_DIR}/missing-baseline.out"

	if BENCH_NIGHTLY_STATE_DIR="${state_dir}" \
		RESULTS_ROOT="${TMP_DIR}/results" \
		bash "${ROOT_DIR}/scripts/bench/nightly-gate.sh" run >"${output}" 2>&1; then
		printf 'expected nightly-gate run to fail without baseline\n' >&2
		return 1
	fi

	fct_assert_contains "${output}" "未找到 nightly baseline"
}

fct_test_seed_persists_latest_go_bench_as_baseline() {
	TMP_DIR="$(mktemp -d)"
	local bin_dir="${TMP_DIR}/bin"
	local state_dir="${TMP_DIR}/state"
	fct_write_stubs "${bin_dir}"

	BENCH_RUN_SCRIPT="${bin_dir}/run.sh" \
		BENCH_NIGHTLY_STATE_DIR="${state_dir}" \
		RESULTS_ROOT="${TMP_DIR}/results" \
		STUB_RUN_ENV_FILE="${TMP_DIR}/run.env" \
		STUB_GO_BENCH_CONTENT="BenchmarkSeed 1 123 ns/op 4 B/op 1 allocs/op" \
		bash "${ROOT_DIR}/scripts/bench/nightly-gate.sh" seed

	fct_assert_file "${state_dir}/baseline-go-bench.txt"
	fct_assert_contains "${state_dir}/baseline-go-bench.txt" "BenchmarkSeed"
	fct_assert_contains "${state_dir}/last-run.env" "status=SEEDED"
	fct_assert_contains "${TMP_DIR}/run.env" "BENCH_COUNT=10"
}

fct_test_run_propagates_gate_failure_without_rotating_baseline() {
	TMP_DIR="$(mktemp -d)"
	local bin_dir="${TMP_DIR}/bin"
	local state_dir="${TMP_DIR}/state"
	local output="${TMP_DIR}/gate-fail.out"
	fct_write_stubs "${bin_dir}"
	mkdir -p "${state_dir}"
	printf 'BenchmarkBaseline 1 100 ns/op 0 B/op 0 allocs/op\n' >"${state_dir}/baseline-go-bench.txt"

	local status=0
	BENCH_CI_GATE_SCRIPT="${bin_dir}/ci-gate.sh" \
		BENCH_NIGHTLY_STATE_DIR="${state_dir}" \
		RESULTS_ROOT="${TMP_DIR}/results" \
		STUB_GATE_ENV_FILE="${TMP_DIR}/gate.env" \
		STUB_GATE_EXIT=17 \
		bash "${ROOT_DIR}/scripts/bench/nightly-gate.sh" run >"${output}" 2>&1 || status=$?

	if [[ "${status}" -ne 17 ]]; then
		printf 'expected exit 17, got %s\n' "${status}" >&2
		return 1
	fi
	fct_assert_contains "${state_dir}/baseline-go-bench.txt" "BenchmarkBaseline"
	fct_assert_contains "${state_dir}/last-run.env" "status=FAILED"
	fct_assert_contains "${TMP_DIR}/gate.env" "BENCH_BASELINE=${state_dir}/baseline-go-bench.txt"
	fct_assert_contains "${TMP_DIR}/gate.env" "BENCH_COUNT=10"
}

fct_test_run_rotates_baseline_after_pass() {
	TMP_DIR="$(mktemp -d)"
	local bin_dir="${TMP_DIR}/bin"
	local state_dir="${TMP_DIR}/state"
	fct_write_stubs "${bin_dir}"
	mkdir -p "${state_dir}"
	printf 'BenchmarkBaseline 1 100 ns/op 0 B/op 0 allocs/op\n' >"${state_dir}/baseline-go-bench.txt"

	BENCH_CI_GATE_SCRIPT="${bin_dir}/ci-gate.sh" \
		BENCH_NIGHTLY_STATE_DIR="${state_dir}" \
		RESULTS_ROOT="${TMP_DIR}/results" \
		STUB_GATE_ENV_FILE="${TMP_DIR}/gate.env" \
		STUB_CURRENT_GO_BENCH_CONTENT="BenchmarkCurrent 1 80 ns/op 0 B/op 0 allocs/op" \
		bash "${ROOT_DIR}/scripts/bench/nightly-gate.sh" run

	fct_assert_contains "${state_dir}/baseline-go-bench.txt" "BenchmarkCurrent"
	fct_assert_contains "${state_dir}/last-run.env" "status=PASSED"
}

fct_test_install_cron_dry_run_prints_schedule() {
	TMP_DIR="$(mktemp -d)"
	local output="${TMP_DIR}/cron.out"

	BENCH_NIGHTLY_CRON_SCHEDULE="7 3 * * *" \
		bash "${ROOT_DIR}/scripts/bench/install-nightly-cron.sh" --dry-run >"${output}"

	fct_assert_contains "${output}" "dry-run: would install"
	fct_assert_contains "${output}" "7 3 * * *"
	fct_assert_contains "${output}" "scripts/bench/nightly-gate.sh run"
}

fct_test_run_requires_seeded_baseline
fct_test_seed_persists_latest_go_bench_as_baseline
fct_test_run_propagates_gate_failure_without_rotating_baseline
fct_test_run_rotates_baseline_after_pass
fct_test_install_cron_dry_run_prints_schedule

printf 'nightly gate tests passed\n'
