#!/usr/bin/env bash
set -euo pipefail

API_BASE_URL="${API_BASE_URL:-http://127.0.0.1:5000}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-33306}"
MYSQL_USER="${MYSQL_USER:-root}"
MYSQL_PWD="${MYSQL_PWD:-root}"
REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
TOKEN="${TOKEN:-}"
USER_ID="${USER_ID:-1}"

if [ -z "${TOKEN}" ]; then
  echo "TOKEN is required" >&2
  exit 1
fi

article_resp=$(curl -sS -X POST "${API_BASE_URL}/v1/content/article/publish" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "title":"publish article",
    "description":"content domain article",
    "cover":"https://example.com/article-cover.png",
    "content":"hello zfeed article",
    "visibility":10
  }')

echo "ARTICLE_RESPONSE=${article_resp}"

video_resp=$(curl -sS -X POST "${API_BASE_URL}/v1/content/video/publish" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{
    "title":"publish video",
    "description":"content domain video",
    "video_url":"https://example.com/video.mp4",
    "cover_url":"https://example.com/video-cover.png",
    "duration":120,
    "visibility":10
  }')

echo "VIDEO_RESPONSE=${video_resp}"

MYSQL_PWD="${MYSQL_PWD}" mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -D zfeed \
  -e "SELECT id,user_id,content_type,status,visibility,published_at FROM zfeed_content ORDER BY id DESC LIMIT 10;"

MYSQL_PWD="${MYSQL_PWD}" mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -D zfeed \
  -e "SELECT id,content_id,title FROM zfeed_article ORDER BY id DESC LIMIT 10;"

MYSQL_PWD="${MYSQL_PWD}" mysql -h"${MYSQL_HOST}" -P"${MYSQL_PORT}" -u"${MYSQL_USER}" -D zfeed \
  -e "SELECT id,content_id,title,origin_url,cover_url,duration,transcode_status FROM zfeed_video ORDER BY id DESC LIMIT 10;"

redis-cli -h "${REDIS_HOST}" ZRANGE "feed:user:publish:${USER_ID}" 0 -1 WITHSCORES
