#!/usr/bin/env bash
set -Eeuo pipefail

# SCRIPT INFO
# Name: test_count_chain.sh
# Purpose: Replay the Day15 count write-chain verification for like/follow.
# Scope: local zfeed repo with docker infra running and local count-rpc listening.

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly DEPLOY_DIR="${ROOT_DIR}/deploy"
readonly ENV_FILE_PATH="${ROOT_DIR}/.env.local"

readonly COUNT_CONSUMER_NAME="count.canal_consumer"
readonly LIKE_INSERT_EVENT_ID="day15-like-verify-insert"
readonly FOLLOW_INSERT_EVENT_ID="day15-follow-verify-insert"
readonly LIKE_UPDATE_EVENT_ID="day15-like-verify-update"
readonly FOLLOW_UPDATE_EVENT_ID="day15-follow-verify-update"

readonly LIKE_TARGET_ID="22001"
readonly FOLLOWING_TARGET_ID="12001"
readonly FOLLOWED_TARGET_ID="32001"
readonly OWNER_ID="32001"

readonly COUNT_VALUE_KEY="count:value:10:10:${LIKE_TARGET_ID}"
readonly OWNER_PROFILE_KEY="count:user:profile:${OWNER_ID}"
readonly FOLLOWING_PROFILE_KEY="count:user:profile:${FOLLOWING_TARGET_ID}"

fct_docker_compose() {
  (
    cd "${DEPLOY_DIR}"
    docker compose --env-file .env -f docker-compose.yml "$@"
  )
}

fct_require_env() {
  if [ ! -f "${ENV_FILE_PATH}" ]; then
    printf 'Missing env file: %s\n' "${ENV_FILE_PATH}" >&2
    exit 1
  fi

  set -a
  . "${ENV_FILE_PATH}"
  set +a
}

fct_require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "${cmd}" >&2
    exit 1
  fi
}

fct_port_from_listen_on() {
  local listen_on="$1"
  printf '%s\n' "${listen_on##*:}"
}

fct_require_count_rpc_running() {
  local count_rpc_port
  count_rpc_port="$(fct_port_from_listen_on "${COUNT_RPC_LISTEN_ON}")"

  if (echo >"/dev/tcp/127.0.0.1/${count_rpc_port}") >/dev/null 2>&1; then
    return 0
  fi

  printf 'count-rpc is not listening on 127.0.0.1:%s\n' "${count_rpc_port}" >&2
  printf 'Start it first with script/start.sh or go run ./app/rpc/count ...\n' >&2
  exit 1
}

fct_mysql_exec() {
  local sql="$1"
  MYSQL_PWD=root mysql -h127.0.0.1 -P"${MYSQL_APP_PORT}" -uroot -N -B -e "${sql}"
}

fct_redis_cmd() {
  redis-cli -p "${REDIS_PORT}" "$@"
}

fct_reset_state() {
  fct_mysql_exec "
    USE zfeed;
    DELETE FROM zfeed_count_value
    WHERE (biz_type, target_type, target_id) IN (
      (10, 10, ${LIKE_TARGET_ID}),
      (41, 20, ${FOLLOWING_TARGET_ID}),
      (40, 20, ${FOLLOWED_TARGET_ID})
    );
    DELETE FROM zfeed_mq_consume_dedup
    WHERE consumer = '${COUNT_CONSUMER_NAME}';
  " >/dev/null

  fct_redis_cmd DEL "${COUNT_VALUE_KEY}" "${OWNER_PROFILE_KEY}" "${FOLLOWING_PROFILE_KEY}" >/dev/null
}

fct_seed_cache_markers() {
  fct_redis_cmd SET "${COUNT_VALUE_KEY}" "sentinel-like" >/dev/null
  fct_redis_cmd SET "${OWNER_PROFILE_KEY}" "sentinel-owner" >/dev/null
  fct_redis_cmd SET "${FOLLOWING_PROFILE_KEY}" "sentinel-following" >/dev/null
}

fct_publish_messages() {
  local payload="$1"
  printf '%s\n' "${payload}" | \
    fct_docker_compose exec -T kafka bash -lc \
      "/opt/bitnami/kafka/bin/kafka-console-producer.sh --bootstrap-server localhost:9092 --topic zfeed-count-canal >/dev/null"
}

fct_wait_consume() {
  sleep 2
}

fct_assert_contains() {
  local haystack="$1"
  local needle="$2"
  local hint="$3"
  if ! grep -Fq "${needle}" <<<"${haystack}"; then
    printf 'Assertion failed: %s\nExpected to find: %s\n' "${hint}" "${needle}" >&2
    printf 'Actual output:\n%s\n' "${haystack}" >&2
    exit 1
  fi
}

fct_assert_empty() {
  local value="$1"
  local hint="$2"
  if [ -n "${value}" ]; then
    printf 'Assertion failed: %s\nExpected empty but got: %s\n' "${hint}" "${value}" >&2
    exit 1
  fi
}

fct_verify_insert_phase() {
  local count_rows
  count_rows="$(
    fct_mysql_exec "
      USE zfeed;
      SELECT CONCAT_WS(',', biz_type, target_type, target_id, owner_id, value, version)
      FROM zfeed_count_value
      WHERE (biz_type, target_type, target_id) IN (
        (10, 10, ${LIKE_TARGET_ID}),
        (41, 20, ${FOLLOWING_TARGET_ID}),
        (40, 20, ${FOLLOWED_TARGET_ID})
      )
      ORDER BY biz_type, target_type, target_id;
    "
  )"

  printf 'Insert phase rows:\n%s\n' "${count_rows}"
  fct_assert_contains "${count_rows}" "10,10,${LIKE_TARGET_ID},${OWNER_ID},1,1" "like insert should land content count"
  fct_assert_contains "${count_rows}" "40,20,${FOLLOWED_TARGET_ID},0,1,1" "follow insert should land followed count"
  fct_assert_contains "${count_rows}" "41,20,${FOLLOWING_TARGET_ID},0,1,1" "follow insert should land following count"
}

fct_verify_update_phase() {
  local count_rows
  local count_value
  local owner_profile
  local following_profile

  count_rows="$(
    fct_mysql_exec "
      USE zfeed;
      SELECT CONCAT_WS(',', biz_type, target_type, target_id, owner_id, value, version)
      FROM zfeed_count_value
      WHERE (biz_type, target_type, target_id) IN (
        (10, 10, ${LIKE_TARGET_ID}),
        (41, 20, ${FOLLOWING_TARGET_ID}),
        (40, 20, ${FOLLOWED_TARGET_ID})
      )
      ORDER BY biz_type, target_type, target_id;
    "
  )"

  count_value="$(fct_redis_cmd GET "${COUNT_VALUE_KEY}")"
  owner_profile="$(fct_redis_cmd GET "${OWNER_PROFILE_KEY}")"
  following_profile="$(fct_redis_cmd GET "${FOLLOWING_PROFILE_KEY}")"

  printf 'Update phase rows:\n%s\n' "${count_rows}"
  printf 'Redis after update:\n'
  printf '  %s => %s\n' "${COUNT_VALUE_KEY}" "${count_value:-<empty>}"
  printf '  %s => %s\n' "${OWNER_PROFILE_KEY}" "${owner_profile:-<empty>}"
  printf '  %s => %s\n' "${FOLLOWING_PROFILE_KEY}" "${following_profile:-<empty>}"

  fct_assert_contains "${count_rows}" "10,10,${LIKE_TARGET_ID},${OWNER_ID},0,2" "like cancel should zero content count"
  fct_assert_contains "${count_rows}" "40,20,${FOLLOWED_TARGET_ID},0,0,2" "follow cancel should zero followed count"
  fct_assert_contains "${count_rows}" "41,20,${FOLLOWING_TARGET_ID},0,0,2" "follow cancel should zero following count"
  fct_assert_empty "${count_value}" "count value cache should be deleted"
  fct_assert_empty "${owner_profile}" "owner profile cache should be deleted"
  fct_assert_empty "${following_profile}" "following profile cache should be deleted"
}

fct_execute_this() {
  local insert_payload
  local update_payload

  fct_require_env
  fct_require_command docker
  fct_require_command mysql
  fct_require_command redis-cli
  fct_require_count_rpc_running

  fct_reset_state
  fct_seed_cache_markers

  insert_payload="$(cat <<EOF
{"id":"${LIKE_INSERT_EVENT_ID}","table":"zfeed_like","type":"INSERT","ts":1775553600000,"data":[{"id":900021,"user_id":12001,"content_id":${LIKE_TARGET_ID},"content_user_id":${OWNER_ID},"status":10,"is_deleted":0}],"old":[]}
{"id":"${FOLLOW_INSERT_EVENT_ID}","table":"zfeed_follow","type":"INSERT","ts":1775553601000,"data":[{"id":900022,"user_id":${FOLLOWING_TARGET_ID},"follow_user_id":${FOLLOWED_TARGET_ID},"status":10,"is_deleted":0}],"old":[]}
EOF
)"
  fct_publish_messages "${insert_payload}"
  fct_wait_consume
  fct_verify_insert_phase

  update_payload="$(cat <<EOF
{"id":"${LIKE_UPDATE_EVENT_ID}","table":"zfeed_like","type":"UPDATE","ts":1775553660000,"data":[{"id":900021,"user_id":12001,"content_id":${LIKE_TARGET_ID},"content_user_id":${OWNER_ID},"status":20,"is_deleted":0}],"old":[{"status":10}]}
{"id":"${FOLLOW_UPDATE_EVENT_ID}","table":"zfeed_follow","type":"UPDATE","ts":1775553661000,"data":[{"id":900022,"user_id":${FOLLOWING_TARGET_ID},"follow_user_id":${FOLLOWED_TARGET_ID},"status":20,"is_deleted":0}],"old":[{"status":10,"is_deleted":0}]}
EOF
)"
  fct_publish_messages "${update_payload}"
  fct_wait_consume
  fct_verify_update_phase

  printf 'Day15 count write-chain verification passed.\n'
}

fct_main() {
  trap 'printf "Script failed at line %s\n" "${LINENO}" >&2' ERR
  fct_execute_this
}

fct_main "$@"
