#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

if [[ ! -f "bench/k6/scenarios/load.js" ]]; then
	printf 'missing k6 load scenario: bench/k6/scenarios/load.js\n' >&2
	exit 1
fi

if ! rg -q 'fct_run_k6 "load"' scripts/bench/run.sh; then
	printf 'run.sh load command should execute the load scenario file directly\n' >&2
	exit 1
fi

if rg -q 'ENABLE_LOG_PIPELINE="\$\{ENABLE_LOG_PIPELINE:-1\}"' scripts/bench/run.sh; then
	printf 'bench start-stack should not enable the optional log pipeline by default\n' >&2
	exit 1
fi

if rg -q 'ENABLE_TRACE_PIPELINE="\$\{ENABLE_TRACE_PIPELINE:-1\}"' scripts/bench/run.sh; then
	printf 'bench start-stack should not enable the optional trace pipeline by default\n' >&2
	exit 1
fi

node --input-type=module <<'EOF'
globalThis.__ENV = {};
const { stressStages } = await import("./bench/k6/config.js");
const maxTarget = Math.max(...stressStages.map((stage) => Number(stage.target)));
if (maxTarget !== 50) {
  console.error(`expected stressStages max target 50, got ${maxTarget}`);
  process.exit(1);
}
EOF
