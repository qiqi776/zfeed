#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEPLOY_DIR="$ROOT_DIR/deploy"
RUNTIME_DIR="$ROOT_DIR/logs/runtime"
ENV_FILE_PATH="$ROOT_DIR/.env.local"

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

fct_port_busy() {
  local port="$1"
  lsof -iTCP:"$port" -sTCP:LISTEN -t >/dev/null 2>&1
}

fct_stop_port_listener() {
  local port="$1"
  local name="$2"
  local pids
  local pid

  pids=$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | sort -u || true)
  if [ -z "$pids" ]; then
    return 0
  fi

  echo "Stopping existing $name listener on port $port..."
  for pid in $pids; do
    kill "$pid" 2>/dev/null || true
  done

  for _ in $(seq 1 20); do
    if ! fct_port_busy "$port"; then
      return 0
    fi
    sleep 1
  done

  pids=$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null | sort -u || true)
  for pid in $pids; do
    kill -9 "$pid" 2>/dev/null || true
  done
}

fct_port_from_listen_on() {
  local listen_on="$1"
  printf '%s\n' "${listen_on##*:}"
}

echo "Stopping local Go services..."
fct_stop_pid_file "$RUNTIME_DIR/front-api.pid"
fct_stop_pid_file "$RUNTIME_DIR/count-rpc.pid"
fct_stop_pid_file "$RUNTIME_DIR/interaction-rpc.pid"
fct_stop_pid_file "$RUNTIME_DIR/content-rpc.pid"
fct_stop_pid_file "$RUNTIME_DIR/user-rpc.pid"

if [ -f "$ENV_FILE_PATH" ]; then
  . "$ENV_FILE_PATH"

  FRONT_API_PORT="${FRONT_API_PORT:-5000}"
  FRONT_PROM_PORT="${PROM_PORT:-9290}"
  CONTENT_PROM_PORT="${CONTENT_PROM_PORT:-9291}"
  INTERACTION_PROM_PORT="${INTERACTION_PROM_PORT:-9293}"
  COUNT_PROM_PORT="${COUNT_PROM_PORT:-9292}"
  USER_PROM_PORT="${USER_PROM_PORT:-9294}"
  USER_RPC_PORT=$(fct_port_from_listen_on "${USER_RPC_LISTEN_ON:-127.0.0.1:5003}")
  CONTENT_RPC_PORT=$(fct_port_from_listen_on "${CONTENT_RPC_LISTEN_ON:-127.0.0.1:5001}")
  INTERACTION_RPC_PORT=$(fct_port_from_listen_on "${INTERACTION_RPC_LISTEN_ON:-127.0.0.1:5002}")
  COUNT_RPC_PORT=$(fct_port_from_listen_on "${COUNT_RPC_LISTEN_ON:-127.0.0.1:5004}")
  XXL_EXECUTOR_PORT=$(fct_port_from_listen_on "${XXL_EXECUTOR_ADDRESS:-127.0.0.1:5005}")

  fct_stop_port_listener "$FRONT_API_PORT" "front-api"
  fct_stop_port_listener "$FRONT_PROM_PORT" "front-api metrics"
  fct_stop_port_listener "$COUNT_RPC_PORT" "count-rpc"
  fct_stop_port_listener "$COUNT_PROM_PORT" "count-rpc metrics"
  fct_stop_port_listener "$INTERACTION_RPC_PORT" "interaction-rpc"
  fct_stop_port_listener "$INTERACTION_PROM_PORT" "interaction-rpc metrics"
  fct_stop_port_listener "$CONTENT_RPC_PORT" "content-rpc"
  fct_stop_port_listener "$CONTENT_PROM_PORT" "content-rpc metrics"
  fct_stop_port_listener "$USER_RPC_PORT" "user-rpc"
  fct_stop_port_listener "$USER_PROM_PORT" "user-rpc metrics"
  if [ "$XXL_EXECUTOR_PORT" != "$CONTENT_RPC_PORT" ]; then
    fct_stop_port_listener "$XXL_EXECUTOR_PORT" "content-rpc xxl executor"
  fi
fi

echo "Stopping Docker infrastructure..."
fct_docker_compose down
