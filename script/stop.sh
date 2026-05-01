#!/usr/bin/env bash
set -Eeuo pipefail

readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly DEPLOY_DIR="${ROOT_DIR}/deploy"
readonly DOWN="${DOWN:-0}"

fct_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    (
      cd "${DEPLOY_DIR}"
      docker compose --env-file .env -f docker-compose.yml "$@"
    )
    return 0
  fi

  if ! command -v powershell.exe >/dev/null 2>&1; then
    echo "docker compose is unavailable and powershell.exe is missing." >&2
    return 1
  fi

  local deploy_dir_win
  deploy_dir_win="$(wslpath -w "${DEPLOY_DIR}")"

  local ps_cmd='cmd /c "pushd '"${deploy_dir_win}"' && docker compose --env-file .env -f docker-compose.yml'
  local arg
  for arg in "$@"; do
    ps_cmd="${ps_cmd} ${arg}"
  done
  ps_cmd="${ps_cmd}\""

  powershell.exe -NoProfile -Command "${ps_cmd}"
}

if [ "${DOWN}" = "1" ]; then
  echo "Stopping Docker backend and infrastructure with container removal..."
  fct_docker_compose down --remove-orphans
else
  echo "Stopping Docker backend and infrastructure without removing containers..."
  fct_docker_compose stop
fi
