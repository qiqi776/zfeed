#!/bin/sh
set -eu

SEARCH_INDEX_ENGINE_ENDPOINT="${SEARCH_INDEX_ENGINE_ENDPOINT:-http://localhost:9200}"
SEARCH_INDEX_CONTENT_INDEX="${SEARCH_INDEX_CONTENT_INDEX:-zfeed_content}"
SEARCH_INDEX_USER_INDEX="${SEARCH_INDEX_USER_INDEX:-zfeed_user}"
SEARCH_INDEX_CONTENT_VERSION="${SEARCH_INDEX_CONTENT_VERSION:-zfeed_content_v1}"
SEARCH_INDEX_USER_VERSION="${SEARCH_INDEX_USER_VERSION:-zfeed_user_v1}"
SEARCH_INDEX_ENGINE_USERNAME="${SEARCH_INDEX_ENGINE_USERNAME:-}"
SEARCH_INDEX_ENGINE_PASSWORD="${SEARCH_INDEX_ENGINE_PASSWORD:-}"

fct_curl() {
  method="${1}"
  path="${2}"
  data="${3:-}"
  if [ -n "${SEARCH_INDEX_ENGINE_USERNAME}" ]; then
    auth="-u ${SEARCH_INDEX_ENGINE_USERNAME}:${SEARCH_INDEX_ENGINE_PASSWORD}"
  else
    auth=""
  fi
  # shellcheck disable=SC2086
  if [ -n "${data}" ]; then
    curl -fsS ${auth} -X "${method}" "${SEARCH_INDEX_ENGINE_ENDPOINT}${path}" -H 'Content-Type: application/json' --data-binary "${data}"
  else
    curl -fsS ${auth} -X "${method}" "${SEARCH_INDEX_ENGINE_ENDPOINT}${path}"
  fi
}

fct_wait_ready() {
  attempt=1
  while [ "${attempt}" -le 60 ]; do
    if fct_curl GET "/_cluster/health" >/dev/null 2>&1; then
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 1
  done
  printf 'search engine is not ready: %s\n' "${SEARCH_INDEX_ENGINE_ENDPOINT}" >&2
  return 1
}

fct_index_exists() {
  fct_curl GET "/${1}" >/dev/null 2>&1
}

fct_create_content_index() {
  if fct_index_exists "${SEARCH_INDEX_CONTENT_VERSION}"; then
    return 0
  fi
  fct_curl PUT "/${SEARCH_INDEX_CONTENT_VERSION}" '{
    "settings": {
      "index.max_ngram_diff": 2,
      "analysis": {
        "tokenizer": {
          "zfeed_ngram_tokenizer": {
            "type": "ngram",
            "min_gram": 1,
            "max_gram": 3,
            "token_chars": ["letter", "digit"]
          }
        },
        "analyzer": {
          "zfeed_text": {
            "type": "custom",
            "tokenizer": "zfeed_ngram_tokenizer",
            "filter": ["lowercase"]
          }
        }
      }
    },
    "mappings": {
      "properties": {
        "content_id": {"type": "long"},
        "content_type": {"type": "integer"},
        "title": {"type": "text", "analyzer": "zfeed_text", "fields": {"keyword": {"type": "keyword", "ignore_above": 256}}},
        "description": {"type": "text", "analyzer": "zfeed_text"},
        "author_id": {"type": "long"},
        "author_name": {"type": "text", "analyzer": "zfeed_text", "fields": {"keyword": {"type": "keyword", "ignore_above": 256}}},
        "author_avatar": {"type": "keyword", "ignore_above": 512},
        "published_at": {"type": "long"},
        "visibility": {"type": "integer"},
        "status": {"type": "integer"},
        "hot_score": {"type": "double"}
      }
    }
  }' >/dev/null
}

fct_create_user_index() {
  if fct_index_exists "${SEARCH_INDEX_USER_VERSION}"; then
    return 0
  fi
  fct_curl PUT "/${SEARCH_INDEX_USER_VERSION}" '{
    "settings": {
      "index.max_ngram_diff": 2,
      "analysis": {
        "tokenizer": {
          "zfeed_ngram_tokenizer": {
            "type": "ngram",
            "min_gram": 1,
            "max_gram": 3,
            "token_chars": ["letter", "digit"]
          }
        },
        "analyzer": {
          "zfeed_text": {
            "type": "custom",
            "tokenizer": "zfeed_ngram_tokenizer",
            "filter": ["lowercase"]
          }
        }
      }
    },
    "mappings": {
      "properties": {
        "user_id": {"type": "long"},
        "nickname": {"type": "text", "analyzer": "zfeed_text", "fields": {"keyword": {"type": "keyword", "ignore_above": 256}}},
        "bio": {"type": "text", "analyzer": "zfeed_text"},
        "mobile_search_field": {"type": "keyword", "ignore_above": 64},
        "status": {"type": "integer"}
      }
    }
  }' >/dev/null
}

fct_switch_alias() {
  alias="${1}"
  target="${2}"
  fct_curl POST "/_aliases" '{"actions":[{"remove":{"index":"*","alias":"'"${alias}"'","must_exist":false}},{"add":{"index":"'"${target}"'","alias":"'"${alias}"'"}}]}' >/dev/null
}

fct_wait_ready
fct_create_content_index
fct_create_user_index
fct_switch_alias "${SEARCH_INDEX_CONTENT_INDEX}" "${SEARCH_INDEX_CONTENT_VERSION}"
fct_switch_alias "${SEARCH_INDEX_USER_INDEX}" "${SEARCH_INDEX_USER_VERSION}"
printf 'search indexes are ready: %s -> %s, %s -> %s\n' \
  "${SEARCH_INDEX_CONTENT_INDEX}" "${SEARCH_INDEX_CONTENT_VERSION}" \
  "${SEARCH_INDEX_USER_INDEX}" "${SEARCH_INDEX_USER_VERSION}"
