#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
readonly ROOT_DIR

readonly DEFAULT_DATA_ROOT="${ROOT_DIR}/bench/data"

fct_usage() {
	cat <<'EOF'
用法：
  scripts/bench/data.sh generate <small|medium|large> [--reset|--append]
  scripts/bench/data.sh cleanup-db [--dry-run|run]

说明：
  生成安全的合成压测 fixture。脚本只写 bench/data/<scale> 文件，不写数据库。
  --reset  重建目标数据目录，默认行为
  --append 追加一批数据，基于当前 CSV 最大 ID 继续生成

  cleanup-db 默认只允许 --dry-run 打印 SQL。真实执行需要安装 mysql 客户端，并设置：
    BENCH_DB_HOST、BENCH_DB_USER、BENCH_DB_NAME、BENCH_CLEANUP_CONFIRM=delete-bench-data
  可选设置：BENCH_DB_PORT、BENCH_DB_PASSWORD、MYSQL_BIN
EOF
}

fct_scale_counts() {
	local scale="${1}"
	case "${scale}" in
	small)
		printf '100 1000 5000\n'
		;;
	medium)
		printf '10000 100000 1000000\n'
		;;
	large)
		printf '100000 1000000 10000000\n'
		;;
	*)
		printf '未知数据规模：%s\n' "${scale}" >&2
		return 2
		;;
	esac
}

fct_max_csv_id() {
	local file="${1}"
	local fallback="${2}"
	if [[ ! -f "${file}" ]]; then
		printf '%s\n' "${fallback}"
		return 0
	fi
	awk -F',' -v fallback="${fallback}" 'NR > 1 && $1 ~ /^[0-9]+$/ { if ($1 > max) max = $1 } END { if (max == "") print fallback; else print max }' "${file}"
}

fct_generate_users() {
	local file="${1}"
	local start_id="${2}"
	local count="${3}"
	local append="${4}"
	if [[ "${append}" != "1" ]]; then
		printf 'user_id,mobile,password,nickname,avatar,bio,email,birthday\n' >"${file}"
	fi
	local index
	for ((index = 0; index < count; index++)); do
		local user_id=$((start_id + index))
		local mobile_suffix
		mobile_suffix="$(printf '%010d' "${user_id}")"
		printf '%d,+861%s,123456Aa!,bench_user_%d,https://example.com/bench/avatar-%d.png,bench user %d,bench_user_%d@example.com,946684800\n' \
			"${user_id}" "${mobile_suffix: -10}" "${user_id}" "${user_id}" "${user_id}" "${user_id}" >>"${file}"
	done
}

fct_generate_contents() {
	local file="${1}"
	local start_id="${2}"
	local count="${3}"
	local first_user_id="${4}"
	local user_count="${5}"
	local append="${6}"
	if [[ "${append}" != "1" ]]; then
		printf 'content_id,author_id,scene,title\n' >"${file}"
	fi
	local index
	for ((index = 0; index < count; index++)); do
		local content_id=$((start_id + index))
		local author_id=$((first_user_id + (index % user_count)))
		local scene="ARTICLE"
		local title_prefix="bench_article"
		if (( index % 10 == 9 )); then
			scene="VIDEO"
			title_prefix="bench_video"
		fi
		printf '%d,%d,%s,%s_%d\n' "${content_id}" "${author_id}" "${scene}" "${title_prefix}" "${content_id}" >>"${file}"
	done
}

fct_generate_follow_edges() {
	local file="${1}"
	local first_user_id="${2}"
	local user_count="${3}"
	local edge_count="${4}"
	local append="${5}"
	if [[ "${append}" != "1" ]]; then
		printf 'follower_id,followee_id\n' >"${file}"
	fi
	local index
	for ((index = 0; index < edge_count; index++)); do
		local follower_id=$((first_user_id + (index % user_count)))
		local followee_id=$((first_user_id + ((index * 17 + 1) % user_count)))
		if [[ "${follower_id}" -eq "${followee_id}" ]]; then
			followee_id=$((first_user_id + ((index + 1) % user_count)))
		fi
		printf '%d,%d\n' "${follower_id}" "${followee_id}" >>"${file}"
	done
}

fct_generate_search_terms() {
	local file="${1}"
	local scale="${2}"
	local append="${3}"
	if [[ "${append}" != "1" ]]; then
		printf 'query,kind\n' >"${file}"
	fi
	cat >>"${file}" <<EOF
bench,common
bench_article,content
bench_video,content
bench_user,user
bench_${scale}_hot,common
unlikely_bench_empty_term,empty
EOF
}

fct_write_tokens() {
	local file="${1}"
	cat >"${file}" <<'EOF'
{
  "description": "Synthetic benchmark token pool placeholder. Generate fresh tokens during setup; do not commit real tokens.",
  "tokens": []
}
EOF
}

fct_write_summary() {
	local file="${1}"
	local scale="${2}"
	local user_count="${3}"
	local content_count="${4}"
	local interaction_count="${5}"
	local first_user_id="${6}"
	local first_content_id="${7}"
	cat >"${file}" <<EOF
# zfeed bench ${scale} 数据摘要

- 生成时间：$(date -Is)
- 数据规模：${scale}
- 用户数：${user_count}
- 内容数：${content_count}
- 互动数：${interaction_count}
- 热点用户 ID：${first_user_id}
- 热点内容 ID：${first_content_id}
- 数据前缀：bench_
- 说明：本目录只包含合成 fixture，不包含真实 token。
EOF
}

fct_generate() {
	local scale="${1}"
	local mode="${2:-reset}"
	local user_count content_count interaction_count
	read -r user_count content_count interaction_count < <(fct_scale_counts "${scale}")

	local data_root="${DATA_ROOT:-${DEFAULT_DATA_ROOT}}"
	local target_dir="${data_root}/${scale}"
	local append=0
	case "${mode}" in
	--reset | reset)
		rm -rf "${target_dir}"
		mkdir -p "${target_dir}"
		;;
	--append | append)
		append=1
		mkdir -p "${target_dir}"
		;;
	*)
		printf '未知模式：%s\n' "${mode}" >&2
		return 2
		;;
	esac

	local users_file="${target_dir}/users.csv"
	local contents_file="${target_dir}/content_ids.csv"
	local follow_file="${target_dir}/follow_edges.csv"
	local search_file="${target_dir}/search_terms.csv"
	local tokens_file="${target_dir}/tokens.json"
	local summary_file="${target_dir}/summary.md"

	local first_user_id=10001
	local first_content_id=900001
	if [[ "${append}" == "1" ]]; then
		first_user_id=$(( $(fct_max_csv_id "${users_file}" 10000) + 1 ))
		first_content_id=$(( $(fct_max_csv_id "${contents_file}" 900000) + 1 ))
	fi

	fct_generate_users "${users_file}" "${first_user_id}" "${user_count}" "${append}"
	fct_generate_contents "${contents_file}" "${first_content_id}" "${content_count}" "${first_user_id}" "${user_count}" "${append}"
	fct_generate_follow_edges "${follow_file}" "${first_user_id}" "${user_count}" "${interaction_count}" "${append}"
	fct_generate_search_terms "${search_file}" "${scale}" "${append}"
	fct_write_tokens "${tokens_file}"
	fct_write_summary "${summary_file}" "${scale}" "${user_count}" "${content_count}" "${interaction_count}" "${first_user_id}" "${first_content_id}"

	printf '压测数据已生成：%s\n' "${target_dir}"
}

fct_cleanup_sql() {
	cat <<'SQL'
START TRANSACTION;

CREATE TEMPORARY TABLE IF NOT EXISTS bench_cleanup_users (
  id BIGINT PRIMARY KEY
) ENGINE=MEMORY;

CREATE TEMPORARY TABLE IF NOT EXISTS bench_cleanup_contents (
  id BIGINT PRIMARY KEY
) ENGINE=MEMORY;

INSERT IGNORE INTO bench_cleanup_users (id)
SELECT id
FROM zfeed_user
WHERE nickname LIKE 'bench\_%'
   OR username LIKE 'bench\_%'
   OR email LIKE 'bench\_%@example.com'
   OR bio LIKE 'bench user %'
   OR bio = 'bench generated user';

INSERT IGNORE INTO bench_cleanup_contents (id)
SELECT id
FROM zfeed_content
WHERE user_id IN (SELECT id FROM bench_cleanup_users);

INSERT IGNORE INTO bench_cleanup_contents (id)
SELECT content_id
FROM zfeed_article
WHERE title LIKE 'bench\_%'
   OR description LIKE 'bench %';

INSERT IGNORE INTO bench_cleanup_contents (id)
SELECT content_id
FROM zfeed_video
WHERE title LIKE 'bench\_%'
   OR description LIKE 'bench %';

DELETE cv
FROM zfeed_count_value AS cv
LEFT JOIN bench_cleanup_contents AS bc
  ON cv.target_type = 10 AND cv.target_id = bc.id
LEFT JOIN bench_cleanup_users AS bu
  ON (cv.target_type = 20 AND cv.target_id = bu.id) OR cv.owner_id = bu.id
WHERE bc.id IS NOT NULL OR bu.id IS NOT NULL;

DELETE c
FROM zfeed_comment AS c
LEFT JOIN bench_cleanup_contents AS bc
  ON c.content_id = bc.id
LEFT JOIN bench_cleanup_users AS bu
  ON c.user_id = bu.id OR c.content_user_id = bu.id OR c.reply_to_user_id = bu.id
WHERE bc.id IS NOT NULL OR bu.id IS NOT NULL;

DELETE l
FROM zfeed_like AS l
LEFT JOIN bench_cleanup_contents AS bc
  ON l.content_id = bc.id
LEFT JOIN bench_cleanup_users AS bu
  ON l.user_id = bu.id OR l.content_user_id = bu.id
WHERE bc.id IS NOT NULL OR bu.id IS NOT NULL;

DELETE f
FROM zfeed_favorite AS f
LEFT JOIN bench_cleanup_contents AS bc
  ON f.content_id = bc.id
LEFT JOIN bench_cleanup_users AS bu
  ON f.user_id = bu.id OR f.content_user_id = bu.id
WHERE bc.id IS NOT NULL OR bu.id IS NOT NULL;

DELETE fo
FROM zfeed_follow AS fo
LEFT JOIN bench_cleanup_users AS bu
  ON fo.user_id = bu.id OR fo.follow_user_id = bu.id
WHERE bu.id IS NOT NULL;

DELETE a
FROM zfeed_article AS a
JOIN bench_cleanup_contents AS bc
  ON a.content_id = bc.id;

DELETE v
FROM zfeed_video AS v
JOIN bench_cleanup_contents AS bc
  ON v.content_id = bc.id;

DELETE c
FROM zfeed_content AS c
JOIN bench_cleanup_contents AS bc
  ON c.id = bc.id;

DELETE u
FROM zfeed_user AS u
JOIN bench_cleanup_users AS bu
  ON u.id = bu.id;

DROP TEMPORARY TABLE IF EXISTS bench_cleanup_contents;
DROP TEMPORARY TABLE IF EXISTS bench_cleanup_users;

COMMIT;
SQL
}

fct_run_cleanup_db() {
	local mode="${1:---dry-run}"
	case "${mode}" in
	--dry-run | dry-run)
		fct_cleanup_sql
		return 0
		;;
	"" | run)
		;;
	*)
		printf '未知 cleanup-db 模式：%s\n' "${mode}" >&2
		return 2
		;;
	esac

	if [[ "${BENCH_CLEANUP_CONFIRM:-}" != "delete-bench-data" ]]; then
		printf '真实清理数据库前必须设置 BENCH_CLEANUP_CONFIRM=delete-bench-data。\n' >&2
		return 2
	fi

	local mysql_bin="${MYSQL_BIN:-mysql}"
	if ! command -v "${mysql_bin}" >/dev/null 2>&1; then
		printf '缺少 mysql 客户端：%s\n' "${mysql_bin}" >&2
		return 1
	fi

	local db_host="${BENCH_DB_HOST:-}"
	local db_port="${BENCH_DB_PORT:-3306}"
	local db_user="${BENCH_DB_USER:-}"
	local db_name="${BENCH_DB_NAME:-}"
	if [[ -z "${db_host}" || -z "${db_user}" || -z "${db_name}" ]]; then
		printf '真实清理数据库前必须设置 BENCH_DB_HOST、BENCH_DB_USER、BENCH_DB_NAME。\n' >&2
		return 2
	fi

	local mysql_args=(
		--host="${db_host}"
		--port="${db_port}"
		--user="${db_user}"
		--database="${db_name}"
		--batch
	)
	if [[ -n "${BENCH_DB_PASSWORD:-}" ]]; then
		fct_cleanup_sql | MYSQL_PWD="${BENCH_DB_PASSWORD}" "${mysql_bin}" "${mysql_args[@]}"
		return 0
	fi

	fct_cleanup_sql | "${mysql_bin}" "${mysql_args[@]}"
}

fct_main() {
	local command="${1:-}"
	case "${command}" in
	-h | --help | help | "")
		fct_usage
		;;
	generate)
		local scale="${2:-}"
		local mode="${3:---reset}"
		if [[ -z "${scale}" ]]; then
			fct_usage >&2
			return 2
		fi
		fct_generate "${scale}" "${mode}"
		;;
	cleanup-db)
		fct_run_cleanup_db "${2:---dry-run}"
		;;
	*)
		printf '未知命令：%s\n' "${command}" >&2
		fct_usage >&2
		return 2
		;;
	esac
}

fct_main "$@"
