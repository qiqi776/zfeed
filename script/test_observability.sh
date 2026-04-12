#!/usr/bin/env bash
set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly ENV_FILE_PATH="${ROOT_DIR}/.env.local"

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

fct_number_gt() {
  local left="$1"
  local right="$2"
  awk -v left="${left}" -v right="${right}" 'BEGIN { exit !(left > right) }'
}

fct_prom_query_scalar() {
  local query="$1"
  local body
  local value

  body="$(curl -fsS --get "http://127.0.0.1:${PROMETHEUS_HOST_PORT}/api/v1/query" --data-urlencode "query=${query}")"
  value="$(printf '%s\n' "${body}" | sed -n 's/.*"value":\[[^,]*,"\([^"]*\)"\].*/\1/p' | head -n 1)"
  if [ -z "${value}" ]; then
    printf '0\n'
    return 0
  fi

  printf '%s\n' "${value}"
}

fct_wait_prom_job() {
  local job="$1"
  local value

  for _ in $(seq 1 30); do
    value="$(fct_prom_query_scalar "sum(up{job=\"${job}\"})")"
    if awk -v value="${value:-0}" 'BEGIN { exit !(value >= 1) }'; then
      return 0
    fi
    sleep 2
  done

  printf 'Prometheus target not ready: %s\n' "${job}" >&2
  return 1
}

fct_require_metrics_endpoint() {
  local url="$1"
  local needle="${2:-}"

  local body
  body="$(curl -fsS "${url}")"
  if [ -z "${body}" ]; then
    printf 'Metrics endpoint check failed: %s returned empty body\n' "${url}" >&2
    exit 1
  fi
  if [ -n "${needle}" ] && ! printf '%s\n' "${body}" | grep -Fq "${needle}"; then
    printf 'Metrics endpoint check failed: %s missing %s\n' "${url}" "${needle}" >&2
    exit 1
  fi
}

fct_json_number() {
  local body="$1"
  local key="$2"
  printf '%s' "${body}" | sed -n "s/.*\"${key}\":\\([0-9][0-9]*\\).*/\\1/p" | head -n 1
}

fct_json_string() {
  local body="$1"
  local key="$2"
  printf '%s' "${body}" | sed -n "s/.*\"${key}\":\"\\([^\"]*\\)\".*/\\1/p" | head -n 1
}

fct_register_user() {
  local mobile="$1"
  local nickname="$2"
  local email="$3"

  curl -fsS -X POST "http://127.0.0.1:${FRONT_API_PORT}/v1/users" \
    -H 'Content-Type: application/json' \
    -d "{
      \"mobile\":\"${mobile}\",
      \"password\":\"123456Aa!\",
      \"nickname\":\"${nickname}\",
      \"avatar\":\"https://example.com/avatar.png\",
      \"bio\":\"observability check\",
      \"gender\":1,
      \"email\":\"${email}\",
      \"birthday\":946684800
    }"
}

fct_get_me() {
  local token="$1"
  curl -fsS "http://127.0.0.1:${FRONT_API_PORT}/v1/users/me" \
    -H "Authorization: Bearer ${token}"
}

fct_query_profile() {
  local viewer_token="$1"
  local user_id="$2"
  curl -fsS "http://127.0.0.1:${FRONT_API_PORT}/v1/user/profile/${user_id}" \
    -H "Authorization: Bearer ${viewer_token}"
}

fct_publish_article() {
  local token="$1"
  local title="$2"
  curl -fsS -X POST "http://127.0.0.1:${FRONT_API_PORT}/v1/content/article/publish" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${token}" \
    -d "{
      \"title\":\"${title}\",
      \"description\":\"observability article\",
      \"cover\":\"https://example.com/cover.png\",
      \"content\":\"hello observability\",
      \"visibility\":10
    }"
}

fct_follow_user() {
  local token="$1"
  local target_user_id="$2"
  curl -fsS -X POST "http://127.0.0.1:${FRONT_API_PORT}/v1/interaction/followings" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${token}" \
    -d "{
      \"target_user_id\":\"${target_user_id}\"
    }"
}

fct_assert_metric_increased() {
  local name="$1"
  local before="$2"
  local after="$3"

  if ! fct_number_gt "${after}" "${before}"; then
    printf 'Metric did not increase: %s before=%s after=%s\n' "${name}" "${before}" "${after}" >&2
    exit 1
  fi
}

fct_require_log_match() {
  local pattern="$1"
  shift
  if ! grep -E -q "${pattern}" "$@"; then
    printf 'Expected log pattern not found: %s in %s\n' "${pattern}" "$*" >&2
    exit 1
  fi
}

fct_execute_this() {
  local front_metrics_url
  local content_metrics_url
  local interaction_metrics_url
  local count_metrics_url
  local user_metrics_url
  local seed
  local author_mobile
  local viewer_mobile
  local author_response
  local viewer_response
  local author_token
  local viewer_token
  local author_id
  local viewer_id
  local publish_response
  local content_id
  local http_before
  local http_after
  local user_rpc_before
  local user_rpc_after
  local content_rpc_before
  local content_rpc_after
  local interaction_rpc_before
  local interaction_rpc_after
  local count_rpc_before
  local count_rpc_after
  local db_before
  local db_after
  local log_root
  local collector_files=0

  fct_require_env
  fct_require_command curl
  fct_require_command grep
  fct_require_command sed
  fct_require_command awk

  front_metrics_url="http://127.0.0.1:${PROM_PORT}/metrics"
  content_metrics_url="http://127.0.0.1:${CONTENT_PROM_PORT}/metrics"
  interaction_metrics_url="http://127.0.0.1:${INTERACTION_PROM_PORT}/metrics"
  count_metrics_url="http://127.0.0.1:${COUNT_PROM_PORT}/metrics"
  user_metrics_url="http://127.0.0.1:${USER_PROM_PORT}/metrics"
  log_root="${ROOT_DIR}/${LOG_PATH}"

  fct_require_metrics_endpoint "${front_metrics_url}"
  fct_require_metrics_endpoint "${content_metrics_url}"
  fct_require_metrics_endpoint "${interaction_metrics_url}"
  fct_require_metrics_endpoint "${count_metrics_url}"
  fct_require_metrics_endpoint "${user_metrics_url}"

  fct_wait_prom_job "zfeed-front"
  fct_wait_prom_job "zfeed-content"
  fct_wait_prom_job "zfeed-interaction"
  fct_wait_prom_job "zfeed-count"
  fct_wait_prom_job "zfeed-user"

  http_before="$(fct_prom_query_scalar 'sum(http_server_requests_code_total)')"
  user_rpc_before="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-user"})')"
  content_rpc_before="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-content"})')"
  interaction_rpc_before="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-interaction"})')"
  count_rpc_before="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-count"})')"
  db_before="$(fct_prom_query_scalar 'sum(zfeed_db_statement_total)')"

  seed="$(date +%s)"
  author_mobile="+86$(printf '1%010d' "${seed}")"
  viewer_mobile="+86$(printf '1%010d' "$((seed + 1))")"

  author_response="$(fct_register_user "${author_mobile}" "obs-author-${seed}" "obs-author-${seed}@example.com")"
  viewer_response="$(fct_register_user "${viewer_mobile}" "obs-viewer-${seed}" "obs-viewer-${seed}@example.com")"

  author_token="$(fct_json_string "${author_response}" "token")"
  viewer_token="$(fct_json_string "${viewer_response}" "token")"
  author_id="$(fct_json_number "${author_response}" "user_id")"
  viewer_id="$(fct_json_number "${viewer_response}" "user_id")"

  if [ -z "${author_token}" ] || [ -z "${viewer_token}" ] || [ -z "${author_id}" ] || [ -z "${viewer_id}" ]; then
    printf 'Failed to parse register responses.\nAUTHOR=%s\nVIEWER=%s\n' "${author_response}" "${viewer_response}" >&2
    exit 1
  fi

  fct_get_me "${author_token}" >/dev/null
  fct_query_profile "${viewer_token}" "${author_id}" >/dev/null
  publish_response="$(fct_publish_article "${author_token}" "obs-article-${seed}")"
  content_id="$(fct_json_number "${publish_response}" "content_id")"
  if [ -z "${content_id}" ]; then
    printf 'Failed to parse publish response: %s\n' "${publish_response}" >&2
    exit 1
  fi
  fct_follow_user "${viewer_token}" "${author_id}" >/dev/null

  sleep 8

  http_after="$(fct_prom_query_scalar 'sum(http_server_requests_code_total)')"
  user_rpc_after="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-user"})')"
  content_rpc_after="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-content"})')"
  interaction_rpc_after="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-interaction"})')"
  count_rpc_after="$(fct_prom_query_scalar 'sum(rpc_server_requests_code_total{job="zfeed-count"})')"
  db_after="$(fct_prom_query_scalar 'sum(zfeed_db_statement_total)')"

  fct_assert_metric_increased "http_server_requests_code_total" "${http_before}" "${http_after}"
  fct_assert_metric_increased "user rpc requests" "${user_rpc_before}" "${user_rpc_after}"
  fct_assert_metric_increased "content rpc requests" "${content_rpc_before}" "${content_rpc_after}"
  fct_assert_metric_increased "interaction rpc requests" "${interaction_rpc_before}" "${interaction_rpc_after}"
  fct_assert_metric_increased "count rpc requests" "${count_rpc_before}" "${count_rpc_after}"
  fct_assert_metric_increased "zfeed_db_statement_total" "${db_before}" "${db_after}"

  fct_require_log_match '/v1/users|/v1/content/article/publish|/v1/interaction/followings' "${log_root}/front-api/access.log"
  fct_require_log_match '/user.UserService/Register|/user.UserService/GetMe|/user.UserService/GetUserProfile' "${log_root}/user-rpc/access.log"
  fct_require_log_match '/content.ContentService/PublishArticle|/content.ContentService/BackfillFollowInbox' "${log_root}/content-rpc/access.log"
  fct_require_log_match '/interaction.FollowService/FollowUser|/interaction.FollowService/GetFollowSummary' "${log_root}/interaction-rpc/access.log"
  fct_require_log_match '"layer":"db"' "${log_root}/front-api/access.log"
  fct_require_log_match '"layer":"db"' "${log_root}/user-rpc/access.log"
  fct_require_log_match '"layer":"db"' "${log_root}/content-rpc/access.log"
  fct_require_log_match '"layer":"db"' "${log_root}/interaction-rpc/access.log"
  fct_require_log_match '"layer":"db"' "${log_root}/count-rpc/access.log"

  sleep 4

  if [ "${ENABLE_LOG_PIPELINE:-0}" = "1" ]; then
    collector_files="$(find "${log_root}/collected" -maxdepth 1 -name '*.ndjson' | wc -l | tr -d ' ')"
    if [ "${collector_files}" -le 0 ]; then
      printf 'Collector archive check failed: ENABLE_LOG_PIPELINE=1 but logs/collected has no ndjson output.\n' >&2
      exit 1
    fi
    fct_require_log_match '"event_kind":"http"' "${log_root}"/collected/front-api-*.ndjson
    fct_require_log_match '"event_kind":"rpc"' "${log_root}"/collected/user-rpc-*.ndjson
    fct_require_log_match '"event_kind":"db"' "${log_root}"/collected/front-api-*.ndjson "${log_root}"/collected/user-rpc-*.ndjson "${log_root}"/collected/content-rpc-*.ndjson "${log_root}"/collected/interaction-rpc-*.ndjson "${log_root}"/collected/count-rpc-*.ndjson
  else
    printf 'Collector archive check skipped: ENABLE_LOG_PIPELINE=0.\n'
  fi

  printf 'Observability verification passed.\n'
  printf '  author_id=%s viewer_id=%s content_id=%s\n' "${author_id}" "${viewer_id}" "${content_id}"
  printf '  http_server_requests_code_total: %s -> %s\n' "${http_before}" "${http_after}"
  printf '  user rpc requests: %s -> %s\n' "${user_rpc_before}" "${user_rpc_after}"
  printf '  content rpc requests: %s -> %s\n' "${content_rpc_before}" "${content_rpc_after}"
  printf '  interaction rpc requests: %s -> %s\n' "${interaction_rpc_before}" "${interaction_rpc_after}"
  printf '  count rpc requests: %s -> %s\n' "${count_rpc_before}" "${count_rpc_after}"
  printf '  zfeed_db_statement_total: %s -> %s\n' "${db_before}" "${db_after}"
  printf '  log evidence roots: %s/front-api %s/user-rpc %s/content-rpc %s/interaction-rpc %s/count-rpc %s/collected\n' \
    "${log_root}" "${log_root}" "${log_root}" "${log_root}" "${log_root}" "${log_root}"
}

fct_main() {
  trap 'printf "Script failed at line %s\n" "${LINENO}" >&2' ERR
  fct_execute_this
}

fct_main "$@"
