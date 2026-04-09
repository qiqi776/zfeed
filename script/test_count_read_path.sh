#!/usr/bin/env bash
set -Eeuo pipefail

# SCRIPT INFO
# Name: test_count_read_path.sh
# Purpose: Replay the count read-path verification for cache hit/miss, batch query, and profile aggregate.
# Scope: local zfeed repo with docker infra running and local count-rpc listening.

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly ENV_FILE_PATH="${ROOT_DIR}/.env.local"
readonly GO_HELPER_PATH="${ROOT_DIR}/script/count_read_probe.go"

readonly COUNT_HIT_TARGET_ID="51001"
readonly COUNT_MISS_TARGET_ID="51002"
readonly BATCH_CACHE_TARGET_ID="51003"
readonly BATCH_DB_TARGET_ID="51004"
readonly BATCH_ZERO_TARGET_ID="51005"
readonly PROFILE_USER_ID="61001"
readonly PROFILE_LIKE_TARGET_ID_1="61011"
readonly PROFILE_LIKE_TARGET_ID_2="61012"
readonly PROFILE_FAVORITE_TARGET_ID="61013"

readonly COUNT_HIT_CACHE_KEY="count:value:10:10:${COUNT_HIT_TARGET_ID}"
readonly COUNT_MISS_CACHE_KEY="count:value:10:10:${COUNT_MISS_TARGET_ID}"
readonly BATCH_CACHE_KEY="count:value:10:10:${BATCH_CACHE_TARGET_ID}"
readonly BATCH_DB_CACHE_KEY="count:value:10:10:${BATCH_DB_TARGET_ID}"
readonly BATCH_ZERO_CACHE_KEY="count:value:40:20:${BATCH_ZERO_TARGET_ID}"
readonly PROFILE_CACHE_KEY="count:user:profile:${PROFILE_USER_ID}"

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
  MYSQL_PWD="${MYSQL_PASSWORD}" mysql -h127.0.0.1 -P"${MYSQL_APP_PORT}" -u"${MYSQL_USER}" -N -B -e "${sql}"
}

fct_redis_cmd() {
  redis-cli -p "${REDIS_PORT}" "$@"
}

fct_go_probe() {
  COUNT_RPC_ADDR="${COUNT_RPC_LISTEN_ON}" GOCACHE=/tmp/go-build go run "${GO_HELPER_PATH}" "$@"
}

fct_assert_eq() {
  local actual="$1"
  local expected="$2"
  local hint="$3"
  if [ "${actual}" != "${expected}" ]; then
    printf 'Assertion failed: %s\nExpected: %s\nActual: %s\n' "${hint}" "${expected}" "${actual}" >&2
    exit 1
  fi
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

fct_reset_state() {
  fct_mysql_exec "
    USE zfeed;
    DELETE FROM zfeed_count_value
    WHERE target_id IN (
      ${COUNT_HIT_TARGET_ID},
      ${COUNT_MISS_TARGET_ID},
      ${BATCH_CACHE_TARGET_ID},
      ${BATCH_DB_TARGET_ID},
      ${BATCH_ZERO_TARGET_ID},
      ${PROFILE_USER_ID},
      ${PROFILE_LIKE_TARGET_ID_1},
      ${PROFILE_LIKE_TARGET_ID_2},
      ${PROFILE_FAVORITE_TARGET_ID}
    );

    INSERT INTO zfeed_count_value
      (biz_type, target_type, target_id, value, version, owner_id, created_at, updated_at)
    VALUES
      (10, 10, ${COUNT_HIT_TARGET_ID}, 3, 1, 71001, NOW(), NOW()),
      (10, 10, ${COUNT_MISS_TARGET_ID}, 7, 1, 71002, NOW(), NOW()),
      (10, 10, ${BATCH_CACHE_TARGET_ID}, 1, 1, 71003, NOW(), NOW()),
      (10, 10, ${BATCH_DB_TARGET_ID}, 4, 1, 71004, NOW(), NOW()),
      (10, 10, ${PROFILE_LIKE_TARGET_ID_1}, 2, 1, ${PROFILE_USER_ID}, NOW(), NOW()),
      (10, 10, ${PROFILE_LIKE_TARGET_ID_2}, 3, 1, ${PROFILE_USER_ID}, NOW(), NOW()),
      (20, 10, ${PROFILE_FAVORITE_TARGET_ID}, 4, 1, ${PROFILE_USER_ID}, NOW(), NOW()),
      (41, 20, ${PROFILE_USER_ID}, 5, 1, 0, NOW(), NOW()),
      (40, 20, ${PROFILE_USER_ID}, 6, 1, 0, NOW(), NOW());
  " >/dev/null

  fct_redis_cmd DEL \
    "${COUNT_HIT_CACHE_KEY}" \
    "${COUNT_MISS_CACHE_KEY}" \
    "${BATCH_CACHE_KEY}" \
    "${BATCH_DB_CACHE_KEY}" \
    "${BATCH_ZERO_CACHE_KEY}" \
    "${PROFILE_CACHE_KEY}" >/dev/null
}

fct_verify_get_count_cache_hit() {
  printf '\n[1/4] Verify GetCount cache hit...\n'
  fct_redis_cmd SET "${COUNT_HIT_CACHE_KEY}" "9" >/dev/null

  local value
  value="$(fct_go_probe get 10 10 "${COUNT_HIT_TARGET_ID}")"
  printf 'GetCount hit result: %s\n' "${value}"
  fct_assert_eq "${value}" "9" "GetCount should prefer Redis hit value"
}

fct_verify_get_count_cache_miss() {
  printf '\n[2/4] Verify GetCount cache miss rebuild...\n'
  fct_redis_cmd DEL "${COUNT_MISS_CACHE_KEY}" >/dev/null

  local value
  local cached
  value="$(fct_go_probe get 10 10 "${COUNT_MISS_TARGET_ID}")"
  cached="$(fct_redis_cmd GET "${COUNT_MISS_CACHE_KEY}")"

  printf 'GetCount miss result: %s\n' "${value}"
  printf 'Rebuilt Redis value: %s\n' "${cached:-<empty>}"
  fct_assert_eq "${value}" "7" "GetCount should fallback to DB on cache miss"
  fct_assert_eq "${cached}" "7" "GetCount should rebuild Redis after cache miss"
}

fct_verify_batch_get_count() {
  printf '\n[3/4] Verify BatchGetCount mixed hit/miss...\n'
  fct_redis_cmd SET "${BATCH_CACHE_KEY}" "11" >/dev/null
  fct_redis_cmd DEL "${BATCH_DB_CACHE_KEY}" "${BATCH_ZERO_CACHE_KEY}" >/dev/null

  local output
  local db_cached
  local zero_cached
  output="$(fct_go_probe batch "10:10:${BATCH_CACHE_TARGET_ID},10:10:${BATCH_DB_TARGET_ID},40:20:${BATCH_ZERO_TARGET_ID}")"
  db_cached="$(fct_redis_cmd GET "${BATCH_DB_CACHE_KEY}")"
  zero_cached="$(fct_redis_cmd GET "${BATCH_ZERO_CACHE_KEY}")"

  printf 'BatchGetCount output:\n%s\n' "${output}"
  printf 'Rebuilt Redis values:\n'
  printf '  %s => %s\n' "${BATCH_DB_CACHE_KEY}" "${db_cached:-<empty>}"
  printf '  %s => %s\n' "${BATCH_ZERO_CACHE_KEY}" "${zero_cached:-<empty>}"

  fct_assert_contains "${output}" "10:10:${BATCH_CACHE_TARGET_ID}=11" "BatchGetCount should use cached value when present"
  fct_assert_contains "${output}" "10:10:${BATCH_DB_TARGET_ID}=4" "BatchGetCount should load DB value for cache miss"
  fct_assert_contains "${output}" "40:20:${BATCH_ZERO_TARGET_ID}=0" "BatchGetCount should return zero for missing DB row"
  fct_assert_eq "${db_cached}" "4" "BatchGetCount should rebuild DB-miss cache entry"
  fct_assert_eq "${zero_cached}" "0" "BatchGetCount should cache zero for missing DB row"
}

fct_verify_user_profile_counts() {
  printf '\n[4/4] Verify GetUserProfileCounts aggregate...\n'
  fct_redis_cmd DEL "${PROFILE_CACHE_KEY}" >/dev/null

  local profile_output
  local following_value
  local followed_value
  local like_sum
  local favorite_sum
  local profile_cached

  profile_output="$(fct_go_probe profile "${PROFILE_USER_ID}")"
  following_value="$(fct_go_probe get 41 20 "${PROFILE_USER_ID}")"
  followed_value="$(fct_go_probe get 40 20 "${PROFILE_USER_ID}")"
  like_sum="$(
    fct_mysql_exec "
      USE zfeed;
      SELECT COALESCE(SUM(value), 0)
      FROM zfeed_count_value
      WHERE biz_type = 10 AND target_type = 10 AND owner_id = ${PROFILE_USER_ID};
    "
  )"
  favorite_sum="$(
    fct_mysql_exec "
      USE zfeed;
      SELECT COALESCE(SUM(value), 0)
      FROM zfeed_count_value
      WHERE biz_type = 20 AND target_type = 10 AND owner_id = ${PROFILE_USER_ID};
    "
  )"
  profile_cached="$(fct_redis_cmd GET "${PROFILE_CACHE_KEY}")"

  printf 'GetUserProfileCounts output:\n%s\n' "${profile_output}"
  printf 'Single-item comparison:\n'
  printf '  following(single) => %s\n' "${following_value}"
  printf '  followed(single) => %s\n' "${followed_value}"
  printf '  like(sum by owner) => %s\n' "${like_sum}"
  printf '  favorite(sum by owner) => %s\n' "${favorite_sum}"
  printf '  %s => %s\n' "${PROFILE_CACHE_KEY}" "${profile_cached:-<empty>}"

  fct_assert_contains "${profile_output}" "following=5" "profile following count should match count row"
  fct_assert_contains "${profile_output}" "followed=6" "profile followed count should match count row"
  fct_assert_contains "${profile_output}" "like=5" "profile like count should sum owner like counts"
  fct_assert_contains "${profile_output}" "favorite=4" "profile favorite count should sum owner favorite counts"
  fct_assert_eq "${following_value}" "5" "single following count should match profile aggregate"
  fct_assert_eq "${followed_value}" "6" "single followed count should match profile aggregate"
  fct_assert_eq "${like_sum}" "5" "DB like sum should match profile aggregate"
  fct_assert_eq "${favorite_sum}" "4" "DB favorite sum should match profile aggregate"
  if [ -z "${profile_cached}" ]; then
    printf 'Assertion failed: user profile cache should be rebuilt\n' >&2
    exit 1
  fi
}

fct_execute_this() {
  fct_require_env
  fct_require_command go
  fct_require_command mysql
  fct_require_command redis-cli
  fct_require_count_rpc_running

  fct_reset_state
  fct_verify_get_count_cache_hit
  fct_verify_get_count_cache_miss
  fct_verify_batch_get_count
  fct_verify_user_profile_counts

  printf '\nCount read-path verification passed.\n'
}

fct_main() {
  trap 'printf "Script failed at line %s\n" "${LINENO}" >&2' ERR
  fct_execute_this
}

fct_main "$@"
