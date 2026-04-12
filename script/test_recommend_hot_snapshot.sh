#!/usr/bin/env bash
set -Eeuo pipefail

readonly API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:5000}"
readonly REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
readonly REDIS_PORT="${REDIS_PORT:-16379}"
readonly SNAPSHOT_ID="${SNAPSHOT_ID:-recommend-demo-snapshot}"
readonly PAGE_SIZE="${PAGE_SIZE:-2}"

readonly MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
readonly MYSQL_PORT="${MYSQL_APP_PORT:-33306}"
readonly MYSQL_USER="${MYSQL_USER:-zfeed}"
readonly MYSQL_PASSWORD="${MYSQL_PASSWORD:-123456}"
readonly MYSQL_DB="${MYSQL_DB:-zfeed}"

fct_redis_cmd() {
  redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" "$@"
}

fct_fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

fct_extract_hot_ids_from_mysql() {
  if ! command -v mysql >/dev/null 2>&1; then
    return 1
  fi

  MYSQL_PWD="${MYSQL_PASSWORD}" mysql \
    -h"${MYSQL_HOST}" \
    -P"${MYSQL_PORT}" \
    -u"${MYSQL_USER}" \
    -D"${MYSQL_DB}" \
    -N -e "SELECT id FROM zfeed_content WHERE status=30 AND visibility=10 AND is_deleted=0 ORDER BY id DESC LIMIT 4;" \
    | tr '\n' ' '
}

fct_split_hot_ids() {
  local raw="$1"
  local arr=()
  local id
  for id in ${raw//,/ }; do
    if [ -n "${id}" ]; then
      arr+=("${id}")
    fi
  done
  printf '%s\n' "${arr[@]}"
}

fct_extract_next_cursor() {
  local body="$1"
  printf '%s' "${body}" | grep -o '"next_cursor":"\{0,1\}[0-9]\+' | head -1 | grep -o '[0-9]\+'
}

fct_extract_content_ids() {
  local body="$1"
  printf '%s' "${body}" | grep -o '"content_id":"\{0,1\}[0-9]\+' | grep -o '[0-9]\+'
}

fct_query_recommend() {
  local cursor="$1"
  local snapshot_id="$2"
  curl -sS -X POST "${API_BASE_URL}/v1/feed/recommend" \
    -H 'Content-Type: application/json' \
    -d "{
      \"cursor\":\"${cursor}\",
      \"page_size\":${PAGE_SIZE},
      \"snapshot_id\":\"${snapshot_id}\"
    }"
}

fct_assert_line_equals() {
  local actual="$1"
  local expected="$2"
  local hint="$3"
  if [ "${actual}" != "${expected}" ]; then
    printf 'Assertion failed: %s\nexpected=%s\nactual=%s\n' "${hint}" "${expected}" "${actual}" >&2
    exit 1
  fi
}

fct_main() {
  local raw_ids="${HOT_CONTENT_IDS:-}"
  local content_ids=()

  if [ -z "${raw_ids}" ]; then
    if ! raw_ids="$(fct_extract_hot_ids_from_mysql)"; then
      fct_fail "HOT_CONTENT_IDS is empty and mysql is unavailable. Set HOT_CONTENT_IDS=id1,id2,id3,id4"
    fi
  fi

  while IFS= read -r id; do
    content_ids+=("${id}")
  done < <(fct_split_hot_ids "${raw_ids}")

  if [ "${#content_ids[@]}" -lt 4 ]; then
    fct_fail "Need at least 4 content ids. current=${raw_ids}"
  fi

  local id1="${content_ids[0]}"
  local id2="${content_ids[1]}"
  local id3="${content_ids[2]}"
  local id4="${content_ids[3]}"

  local snapshot_key="feed:hot:global:snap:${SNAPSHOT_ID}"

  printf '[1/4] write snapshot key and latest pointer\n'
  fct_redis_cmd DEL "${snapshot_key}" >/dev/null
  fct_redis_cmd ZADD "${snapshot_key}" \
    3.800 "${id1}" \
    2.600 "${id2}" \
    1.900 "${id3}" \
    1.200 "${id4}" >/dev/null
  fct_redis_cmd SET "feed:hot:global:latest" "${SNAPSHOT_ID}" >/dev/null

  printf '[2/4] request first page\n'
  local first_resp
  first_resp="$(fct_query_recommend "" "${SNAPSHOT_ID}")"
  printf 'FIRST_RESPONSE=%s\n' "${first_resp}"

  local first_ids
  first_ids="$(fct_extract_content_ids "${first_resp}" | tr '\n' ' ' | sed 's/[[:space:]]\+$//')"
  fct_assert_line_equals "${first_ids}" "${id1} ${id2}" "first page order should match snapshot score order"

  local next_cursor
  next_cursor="$(fct_extract_next_cursor "${first_resp}")"
  if [ -z "${next_cursor}" ]; then
    fct_fail "first page next_cursor is empty"
  fi

  printf '[3/4] request second page\n'
  local second_resp
  second_resp="$(fct_query_recommend "${next_cursor}" "${SNAPSHOT_ID}")"
  printf 'SECOND_RESPONSE=%s\n' "${second_resp}"

  local second_ids
  second_ids="$(fct_extract_content_ids "${second_resp}" | tr '\n' ' ' | sed 's/[[:space:]]\+$//')"
  fct_assert_line_equals "${second_ids}" "${id3} ${id4}" "second page order should match snapshot score order"

  printf '[4/5] verify fallback behavior when snapshot_id not found\n'
  local fallback_resp
  fallback_resp="$(fct_query_recommend "" "snapshot-not-exists")"
  printf 'FALLBACK_RESPONSE=%s\n' "${fallback_resp}"
  if ! printf '%s' "${fallback_resp}" | grep -q '"snapshot_id":"'"${SNAPSHOT_ID}"'"'; then
    fct_fail "fallback should resolve to latest snapshot id ${SNAPSHOT_ID}"
  fi

  printf '[5/5] verify error when snapshot/latest/global are all missing\n'
  fct_redis_cmd DEL "${snapshot_key}" >/dev/null
  fct_redis_cmd DEL "feed:hot:global:latest" >/dev/null
  fct_redis_cmd DEL "feed:hot:global" >/dev/null
  local miss_resp
  miss_resp="$(fct_query_recommend "" "snapshot-not-exists")"
  printf 'ALL_MISSING_RESPONSE=%s\n' "${miss_resp}"
  if ! printf '%s' "${miss_resp}" | grep -q '热榜缓存不存在'; then
    fct_fail "expected error message 热榜缓存不存在 when all snapshot sources are missing"
  fi

  printf '\nRecommend snapshot verification passed.\n'
}

fct_main "$@"
