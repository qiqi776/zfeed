#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DEPLOY_DIR="$ROOT_DIR/deploy"
ENV_FILE_PATH="$ROOT_DIR/.env.local"
ENV_TEMPLATE_PATH="$ROOT_DIR/.env.local.example"
LOG_DIR="$ROOT_DIR/logs"
RUNTIME_DIR="$LOG_DIR/runtime"
USER_RPC_PID_FILE="$RUNTIME_DIR/user-rpc.pid"
FRONT_API_PID_FILE="$RUNTIME_DIR/front-api.pid"
USER_RPC_LOG="$LOG_DIR/user-rpc.log"
FRONT_API_LOG="$LOG_DIR/front-api.log"

fct_require_env_file() {
  if [ -f "$ENV_FILE_PATH" ]; then
    return 0
  fi

  if [ ! -f "$ENV_TEMPLATE_PATH" ]; then
    echo "Missing env template: $ENV_TEMPLATE_PATH" >&2
    exit 1
  fi

  cp "$ENV_TEMPLATE_PATH" "$ENV_FILE_PATH"
  echo "Created $ENV_FILE_PATH from template. Review values if your local ports differ."
}

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

fct_port_busy() {
  local port="$1"
  lsof -iTCP:"$port" -sTCP:LISTEN -t >/dev/null 2>&1
}

fct_wait_for_port() {
  local port="$1"
  local name="$2"

  for _ in $(seq 1 120); do
    if (echo >"/dev/tcp/127.0.0.1/$port") >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "Timed out waiting for $name on 127.0.0.1:$port" >&2
  return 1
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

mkdir -p "$LOG_DIR" "$RUNTIME_DIR"
fct_require_env_file
export ENV_FILE="$ENV_FILE_PATH"

cd "$ROOT_DIR"

echo "Starting infrastructure via Docker Compose..."
fct_docker_compose up -d etcd redis mysql kafka canal

echo "Waiting for infrastructure ports..."
fct_wait_for_port 2379 "etcd"
fct_wait_for_port 6379 "redis"
fct_wait_for_port 33306 "mysql"

fct_stop_pid_file "$USER_RPC_PID_FILE"
fct_stop_pid_file "$FRONT_API_PID_FILE"

if fct_port_busy 5003; then
  echo "Port 5003 is already in use. Stop the existing process before starting user-rpc." >&2
  exit 1
fi

if fct_port_busy 5000; then
  echo "Port 5000 is already in use. Stop the existing process before starting front-api." >&2
  exit 1
fi

echo "Starting user-rpc locally..."
nohup env ENV_FILE="$ENV_FILE" go run ./app/rpc/user -f app/rpc/user/etc/user.yaml >"$USER_RPC_LOG" 2>&1 &
echo $! >"$USER_RPC_PID_FILE"
if ! fct_wait_for_port 5003 "user-rpc"; then
  tail -n 50 "$USER_RPC_LOG" >&2 || true
  exit 1
fi

echo "Starting front-api locally..."
nohup env ENV_FILE="$ENV_FILE" go run ./app/front -f app/front/etc/front-api.yaml >"$FRONT_API_LOG" 2>&1 &
echo $! >"$FRONT_API_PID_FILE"
if ! fct_wait_for_port 5000 "front-api"; then
  tail -n 50 "$FRONT_API_LOG" >&2 || true
  exit 1
fi

echo "zfeed local stack is ready."
echo "  ENV_FILE: $ENV_FILE"
echo "  user-rpc log: $USER_RPC_LOG"
echo "  front-api log: $FRONT_API_LOG"
echo "  stop command: $ROOT_DIR/script/stop.sh"
