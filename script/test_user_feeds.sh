#!/usr/bin/env bash
set -Eeuo pipefail

# SCRIPT INFO
# Name: test_user_feeds.sh
# Purpose: Verify user publish feed and user favorite feed with cache rebuild.
# Scope: local zfeed services running on localhost with valid author/viewer auth tokens.

readonly API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:5000}"
readonly REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
readonly REDIS_PORT="${REDIS_PORT:-16379}"
readonly AUTHOR_ID="${AUTHOR_ID:-}"
readonly AUTHOR_TOKEN="${AUTHOR_TOKEN:-${TOKEN:-}}"
readonly VIEWER_ID="${VIEWER_ID:-}"
readonly VIEWER_TOKEN="${VIEWER_TOKEN:-${AUTHOR_TOKEN}}"
readonly ARTICLE_TITLE="${ARTICLE_TITLE:-publish feed article}"
readonly ARTICLE_DESCRIPTION="${ARTICLE_DESCRIPTION:-publish feed verify}"
readonly ARTICLE_COVER_URL="${ARTICLE_COVER_URL:-https://example.com/publish-feed-article-cover.png}"
readonly ARTICLE_CONTENT="${ARTICLE_CONTENT:-hello publish feed}"

fct_require_env() {
  if [ -z "${AUTHOR_ID}" ]; then
    printf 'AUTHOR_ID is required.\n' >&2
    exit 1
  fi

  if [ -z "${AUTHOR_TOKEN}" ]; then
    printf 'AUTHOR_TOKEN is required. TOKEN may be used as a fallback.\n' >&2
    exit 1
  fi

  if [ -z "${VIEWER_ID}" ]; then
    printf 'VIEWER_ID is required.\n' >&2
    exit 1
  fi
}

fct_redis_cmd() {
  redis-cli -h "${REDIS_HOST}" -p "${REDIS_PORT}" "$@"
}

fct_extract_json_number() {
  local body="$1"
  local key="$2"
  printf '%s' "${body}" | grep -o "\"${key}\":[0-9]*" | head -1 | cut -d: -f2
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

fct_publish_article() {
  curl -sS -X POST "${API_BASE_URL}/v1/content/article/publish" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${AUTHOR_TOKEN}" \
    -d "{
      \"title\":\"${ARTICLE_TITLE}\",
      \"description\":\"${ARTICLE_DESCRIPTION}\",
      \"cover\":\"${ARTICLE_COVER_URL}\",
      \"content\":\"${ARTICLE_CONTENT}\",
      \"visibility\":10
    }"
}

fct_query_user_publish_feed() {
  curl -sS -X POST "${API_BASE_URL}/v1/feed/user/publish" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${AUTHOR_TOKEN}" \
    -d "{
      \"user_id\":\"${AUTHOR_ID}\",
      \"cursor\":\"\",
      \"page_size\":10
    }"
}

fct_favorite_content() {
  local content_id="$1"
  curl -sS -X POST "${API_BASE_URL}/v1/interaction/favorite" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${VIEWER_TOKEN}" \
    -d "{
      \"content_id\":\"${content_id}\",
      \"content_user_id\":\"${AUTHOR_ID}\",
      \"scene\":\"ARTICLE\"
    }"
}

fct_query_user_favorite_feed() {
  curl -sS -X POST "${API_BASE_URL}/v1/feed/user/favorite" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${VIEWER_TOKEN}" \
    -d "{
      \"user_id\":\"${VIEWER_ID}\",
      \"cursor\":\"\",
      \"page_size\":10
    }"
}

fct_execute_this() {
  local publish_response
  local favorite_response
  local publish_feed_response
  local favorite_feed_first_response
  local favorite_feed_hit_response
  local publish_feed_after_rebuild
  local favorite_feed_after_rebuild
  local publish_zset_before_rebuild
  local favorite_zset_after_first_query
  local publish_zset_after_rebuild
  local favorite_zset_after_rebuild
  local publish_key
  local favorite_key
  local content_id

  fct_require_env

  publish_key="feed:user:publish:${AUTHOR_ID}"
  favorite_key="feed:user:favorite:${VIEWER_ID}"

  printf '[1/6] publish one article\n'
  publish_response="$(fct_publish_article)"
  printf 'PUBLISH_RESPONSE=%s\n' "${publish_response}"

  content_id="$(fct_extract_json_number "${publish_response}" "content_id")"
  if [ -z "${content_id}" ]; then
    printf 'Failed to extract content_id from publish response.\n' >&2
    exit 1
  fi

  printf '\n[2/6] query user publish feed\n'
  publish_feed_response="$(fct_query_user_publish_feed)"
  publish_zset_before_rebuild="$(fct_redis_cmd ZRANGE "${publish_key}" 0 -1 WITHSCORES)"
  printf 'USER_PUBLISH_FEED_RESPONSE=%s\n' "${publish_feed_response}"
  printf 'PUBLISH_ZSET_BEFORE_REBUILD=%s\n' "${publish_zset_before_rebuild}"
  fct_assert_contains "${publish_feed_response}" "\"content_id\":\"${content_id}\"" "user publish feed should include the newly published content"
  fct_assert_contains "${publish_zset_before_rebuild}" "${content_id}" "publish feed zset should contain the new content id"

  printf '\n[3/6] favorite the published content\n'
  favorite_response="$(fct_favorite_content "${content_id}")"
  printf 'FAVORITE_RESPONSE=%s\n' "${favorite_response}"

  printf '\n[4/6] query user favorite feed twice\n'
  favorite_feed_first_response="$(fct_query_user_favorite_feed)"
  favorite_zset_after_first_query="$(fct_redis_cmd ZRANGE "${favorite_key}" 0 -1 WITHSCORES)"
  favorite_feed_hit_response="$(fct_query_user_favorite_feed)"
  printf 'USER_FAVORITE_FEED_FIRST=%s\n' "${favorite_feed_first_response}"
  printf 'FAVORITE_ZSET_AFTER_FIRST_QUERY=%s\n' "${favorite_zset_after_first_query}"
  printf 'USER_FAVORITE_FEED_HIT=%s\n' "${favorite_feed_hit_response}"
  fct_assert_contains "${favorite_feed_first_response}" "\"content_id\":\"${content_id}\"" "user favorite feed should include the favorited content"
  fct_assert_contains "${favorite_zset_after_first_query}" "${content_id}" "favorite feed zset should contain the favorited content id"

  printf '\n[5/6] delete feed caches\n'
  fct_redis_cmd DEL "${publish_key}" "${favorite_key}" >/dev/null

  printf '\n[6/6] query both feeds after cache rebuild\n'
  publish_feed_after_rebuild="$(fct_query_user_publish_feed)"
  favorite_feed_after_rebuild="$(fct_query_user_favorite_feed)"
  publish_zset_after_rebuild="$(fct_redis_cmd ZRANGE "${publish_key}" 0 -1 WITHSCORES)"
  favorite_zset_after_rebuild="$(fct_redis_cmd ZRANGE "${favorite_key}" 0 -1 WITHSCORES)"
  printf 'USER_PUBLISH_FEED_AFTER_REBUILD=%s\n' "${publish_feed_after_rebuild}"
  printf 'USER_FAVORITE_FEED_AFTER_REBUILD=%s\n' "${favorite_feed_after_rebuild}"
  printf 'PUBLISH_ZSET_AFTER_REBUILD=%s\n' "${publish_zset_after_rebuild}"
  printf 'FAVORITE_ZSET_AFTER_REBUILD=%s\n' "${favorite_zset_after_rebuild}"
  fct_assert_contains "${publish_feed_after_rebuild}" "\"content_id\":\"${content_id}\"" "user publish feed should rebuild from DB after cache deletion"
  fct_assert_contains "${favorite_feed_after_rebuild}" "\"content_id\":\"${content_id}\"" "user favorite feed should rebuild from relation data after cache deletion"
  fct_assert_contains "${publish_zset_after_rebuild}" "${content_id}" "publish feed zset should be repopulated after rebuild"
  fct_assert_contains "${favorite_zset_after_rebuild}" "${content_id}" "favorite feed zset should be repopulated after rebuild"

  printf '\nUser feed verification passed. content_id=%s author_id=%s viewer_id=%s\n' "${content_id}" "${AUTHOR_ID}" "${VIEWER_ID}"
}

fct_main() {
  trap 'printf "Script failed at line %s\n" "${LINENO}" >&2' ERR
  fct_execute_this
}

fct_main "$@"
