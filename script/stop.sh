#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEPLOY_DIR="$ROOT_DIR/deploy"
RUNTIME_DIR="$ROOT_DIR/logs/runtime"

fct_docker_compose() {
  if docker compose version >/dev/null 2>&1; then
    (
      cd "$DEPLOY_DIR"
      docker compose --env-file .env -f docker-compose.yml "$@"
    )
    return 0
  fi

  if ! command -v powershell.exe >/dev/null 2>&1; then
    echo "docker compose is unavailable and powershell.exe is missing." >&2
    return 1
  fi

  local deploy_dir_win
  deploy_dir_win=$(wslpath -w "$DEPLOY_DIR")

  local ps_cmd="Set-Location '$deploy_dir_win'; docker compose --env-file .env -f docker-compose.yml"
  local arg
  for arg in "$@"; do
    ps_cmd="$ps_cmd '$arg'"
  done

  powershell.exe -NoProfile -Command "$ps_cmd"
}

fct_stop_pid_file() {
  local pid_file="$1"

  if [ ! -f "$pid_file" ]; then
    return 0
  fi

  local pid
  pid=$(cat "$pid_file")
  if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
  fi
  rm -f "$pid_file"
}

echo "Stopping local Go services..."
fct_stop_pid_file "$RUNTIME_DIR/front-api.pid"
fct_stop_pid_file "$RUNTIME_DIR/count-rpc.pid"
fct_stop_pid_file "$RUNTIME_DIR/interaction-rpc.pid"
fct_stop_pid_file "$RUNTIME_DIR/content-rpc.pid"
fct_stop_pid_file "$RUNTIME_DIR/user-rpc.pid"

echo "Stopping Docker infrastructure..."
fct_docker_compose down
