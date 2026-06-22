package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"zfeed/pkg/mobilex"
)

const (
	benchPasswordSalt = "bench_salt_v1"
	benchPasswordHash = "$2a$10$gR.X.yiu/HFKppspOrNfwO4o.DrKTNNzKaT5WRDWkg.fk9XnKtP/y"
)

type csvRow map[string]string

type fixtureData struct {
	Users       []csvRow
	Contents    []csvRow
	FollowEdges []csvRow
}

type followPair struct {
	FollowerID int64
	FolloweeID int64
}

type verifyItem struct {
	Name     string
	Expected int
	Actual   int
}

type verifyReport struct {
	Items []verifyItem
}

func main() {
	dataDir := flag.String("data", "bench/data/small", "benchmark fixture directory")
	mode := flag.String("mode", "seed-sql", "mode: seed-sql or verify-db")
	dsn := flag.String("dsn", "", "mysql DSN for verify-db mode")
	output := flag.String("output", "", "output SQL path, default stdout")
	flag.Parse()

	switch *mode {
	case "seed-sql":
		sqlText, err := buildSeedSQL(*dataDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "生成 bench seed SQL 失败：%v\n", err)
			os.Exit(1)
		}
		if *output == "" {
			fmt.Print(sqlText)
			return
		}
		if err := os.WriteFile(*output, []byte(sqlText), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "写入 bench seed SQL 失败：%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("bench seed SQL 已生成：%s\n", *output)
	case "verify-db":
		if strings.TrimSpace(*dsn) == "" {
			fmt.Fprintln(os.Stderr, "verify-db 模式必须提供 --dsn")
			os.Exit(2)
		}
		report, err := verifyFixtureDB(context.Background(), *dataDir, *dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "校验 bench fixture 数据库失败：%v\n", err)
			os.Exit(1)
		}
		printVerifyReport(os.Stdout, report)
		if err := report.Validate(); err != nil {
			fmt.Fprintf(os.Stderr, "bench fixture 数据库校验未通过：%v\n", err)
			os.Exit(1)
		}
		fmt.Println("bench fixture 数据库校验通过")
	default:
		fmt.Fprintf(os.Stderr, "未知 mode：%s\n", *mode)
		os.Exit(2)
	}
}

func buildSeedSQL(dataDir string) (string, error) {
	fixture, err := loadFixtureData(dataDir)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	out.WriteString("START TRANSACTION;\n\n")
	renderUsers(&out, fixture.Users)
	renderContents(&out, fixture.Contents)
	renderArticles(&out, fixture.Contents)
	renderVideos(&out, fixture.Contents)
	renderFollowEdges(&out, fixture.FollowEdges)
	out.WriteString("COMMIT;\n")
	return out.String(), nil
}

func loadFixtureData(dataDir string) (fixtureData, error) {
	users, err := readCSVRows(filepath.Join(dataDir, "users.csv"))
	if err != nil {
		return fixtureData{}, err
	}
	contents, err := readCSVRows(filepath.Join(dataDir, "content_ids.csv"))
	if err != nil {
		return fixtureData{}, err
	}
	followEdges, err := readCSVRows(filepath.Join(dataDir, "follow_edges.csv"))
	if err != nil {
		return fixtureData{}, err
	}
	return fixtureData{Users: users, Contents: contents, FollowEdges: followEdges}, nil
}

func readCSVRows(path string) ([]csvRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	for index := range headers {
		headers[index] = strings.TrimSpace(headers[index])
	}

	var rows []csvRow
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		row := csvRow{}
		for index, header := range headers {
			if index < len(record) {
				row[header] = strings.TrimSpace(record[index])
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func renderUsers(out *strings.Builder, rows []csvRow) {
	if len(rows) == 0 {
		return
	}
	out.WriteString("INSERT INTO `zfeed_user` (`id`, `username`, `nickname`, `avatar`, `bio`, `mobile`, `email`, `password_hash`, `password_salt`, `gender`, `birthday`, `status`, `is_deleted`, `created_by`, `updated_by`)\nVALUES\n")
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		userID := rowInt(row, "user_id")
		mobile := mobilex.Normalize(row["mobile"])
		birthday := sqlDateFromUnix(row["birthday"])
		values = append(values, fmt.Sprintf("  (%d, %s, %s, %s, %s, %s, %s, %s, %s, 0, %s, 10, 0, %d, %d)",
			userID,
			sqlString(mobile),
			sqlString(row["nickname"]),
			sqlString(row["avatar"]),
			sqlString(row["bio"]),
			sqlString(mobile),
			sqlString(row["email"]),
			sqlString(benchPasswordHash),
			sqlString(benchPasswordSalt),
			birthday,
			userID,
			userID,
		))
	}
	out.WriteString(strings.Join(values, ",\n"))
	out.WriteString("\nON DUPLICATE KEY UPDATE `nickname` = VALUES(`nickname`), `avatar` = VALUES(`avatar`), `bio` = VALUES(`bio`), `password_hash` = VALUES(`password_hash`), `password_salt` = VALUES(`password_salt`), `status` = VALUES(`status`), `is_deleted` = 0, `updated_by` = VALUES(`updated_by`);\n\n")
}

func renderContents(out *strings.Builder, rows []csvRow) {
	if len(rows) == 0 {
		return
	}
	out.WriteString("INSERT INTO `zfeed_content` (`id`, `user_id`, `content_type`, `status`, `visibility`, `like_count`, `favorite_count`, `comment_count`, `hot_score`, `published_at`, `is_deleted`, `created_by`, `updated_by`)\nVALUES\n")
	values := make([]string, 0, len(rows))
	for _, row := range rows {
		contentID := rowInt(row, "content_id")
		authorID := rowInt(row, "author_id")
		contentType := contentTypeValue(row["scene"])
		values = append(values, fmt.Sprintf("  (%d, %d, %d, 30, 10, 0, 0, 0, 0, NOW(), 0, %d, %d)",
			contentID,
			authorID,
			contentType,
			authorID,
			authorID,
		))
	}
	out.WriteString(strings.Join(values, ",\n"))
	out.WriteString("\nON DUPLICATE KEY UPDATE `user_id` = VALUES(`user_id`), `content_type` = VALUES(`content_type`), `status` = VALUES(`status`), `visibility` = VALUES(`visibility`), `published_at` = VALUES(`published_at`), `is_deleted` = 0, `updated_by` = VALUES(`updated_by`);\n\n")
}

func renderArticles(out *strings.Builder, rows []csvRow) {
	var values []string
	for _, row := range rows {
		if contentTypeValue(row["scene"]) != 10 {
			continue
		}
		contentID := rowInt(row, "content_id")
		title := row["title"]
		values = append(values, fmt.Sprintf("  (%d, %s, %s, %s, %s, 0)",
			contentID,
			sqlString(title),
			sqlString("bench seeded article"),
			sqlString("https://example.com/bench/cover.png"),
			sqlString("hello benchmark"),
		))
	}
	if len(values) == 0 {
		return
	}
	out.WriteString("INSERT INTO `zfeed_article` (`content_id`, `title`, `description`, `cover`, `content`, `is_deleted`)\nVALUES\n")
	out.WriteString(strings.Join(values, ",\n"))
	out.WriteString("\nON DUPLICATE KEY UPDATE `title` = VALUES(`title`), `description` = VALUES(`description`), `cover` = VALUES(`cover`), `content` = VALUES(`content`), `is_deleted` = 0;\n\n")
}

func renderVideos(out *strings.Builder, rows []csvRow) {
	var values []string
	for _, row := range rows {
		if contentTypeValue(row["scene"]) != 20 {
			continue
		}
		contentID := rowInt(row, "content_id")
		title := row["title"]
		values = append(values, fmt.Sprintf("  (%d, %s, %s, %s, %s, 30, 10, 0)",
			contentID,
			sqlString(title),
			sqlString("bench seeded video"),
			sqlString("https://example.com/bench/video.mp4"),
			sqlString("https://example.com/bench/cover.png"),
		))
	}
	if len(values) == 0 {
		return
	}
	out.WriteString("INSERT INTO `zfeed_video` (`content_id`, `title`, `description`, `origin_url`, `cover_url`, `duration`, `transcode_status`, `is_deleted`)\nVALUES\n")
	out.WriteString(strings.Join(values, ",\n"))
	out.WriteString("\nON DUPLICATE KEY UPDATE `title` = VALUES(`title`), `description` = VALUES(`description`), `origin_url` = VALUES(`origin_url`), `cover_url` = VALUES(`cover_url`), `duration` = VALUES(`duration`), `transcode_status` = VALUES(`transcode_status`), `is_deleted` = 0;\n\n")
}

func renderFollowEdges(out *strings.Builder, rows []csvRow) {
	pairs := uniqueFollowPairs(rows)
	if len(pairs) == 0 {
		return
	}
	out.WriteString("INSERT INTO `zfeed_follow` (`user_id`, `follow_user_id`, `status`, `version`, `is_deleted`, `created_by`, `updated_by`)\nVALUES\n")
	values := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		values = append(values, fmt.Sprintf("  (%d, %d, 10, 1, 0, %d, %d)",
			pair.FollowerID,
			pair.FolloweeID,
			pair.FollowerID,
			pair.FollowerID,
		))
	}
	out.WriteString(strings.Join(values, ",\n"))
	out.WriteString("\nON DUPLICATE KEY UPDATE `status` = VALUES(`status`), `version` = VALUES(`version`), `is_deleted` = 0, `updated_by` = VALUES(`updated_by`);\n\n")
}

func verifyFixtureDB(ctx context.Context, dataDir string, dsn string) (verifyReport, error) {
	fixture, err := loadFixtureData(dataDir)
	if err != nil {
		return verifyReport{}, err
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return verifyReport{}, err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return verifyReport{}, err
	}

	userIDs := uniqueRowIDs(fixture.Users, "user_id")
	contentIDs := uniqueRowIDs(fixture.Contents, "content_id")
	articleIDs, videoIDs := contentIDsByType(fixture.Contents)
	followPairs := uniqueFollowPairs(fixture.FollowEdges)

	items := []verifyItem{}
	userCount, err := countRowsByIDs(ctx, db, "zfeed_user", "id", userIDs, " AND `status` = 10 AND `is_deleted` = 0")
	if err != nil {
		return verifyReport{}, fmt.Errorf("校验用户 fixture: %w", err)
	}
	items = append(items, verifyItem{Name: "users", Expected: len(userIDs), Actual: userCount})

	contentCount, err := countRowsByIDs(ctx, db, "zfeed_content", "id", contentIDs, " AND `is_deleted` = 0")
	if err != nil {
		return verifyReport{}, fmt.Errorf("校验内容 fixture: %w", err)
	}
	items = append(items, verifyItem{Name: "contents", Expected: len(contentIDs), Actual: contentCount})

	articleCount, err := countRowsByIDs(ctx, db, "zfeed_article", "content_id", articleIDs, " AND `is_deleted` = 0")
	if err != nil {
		return verifyReport{}, fmt.Errorf("校验文章 fixture: %w", err)
	}
	items = append(items, verifyItem{Name: "articles", Expected: len(articleIDs), Actual: articleCount})

	videoCount, err := countRowsByIDs(ctx, db, "zfeed_video", "content_id", videoIDs, " AND `is_deleted` = 0")
	if err != nil {
		return verifyReport{}, fmt.Errorf("校验视频 fixture: %w", err)
	}
	items = append(items, verifyItem{Name: "videos", Expected: len(videoIDs), Actual: videoCount})

	followCount, err := countRowsByFollowPairs(ctx, db, followPairs)
	if err != nil {
		return verifyReport{}, fmt.Errorf("校验关注 fixture: %w", err)
	}
	items = append(items, verifyItem{Name: "follow_edges", Expected: len(followPairs), Actual: followCount})

	return verifyReport{Items: items}, nil
}

func printVerifyReport(out io.Writer, report verifyReport) {
	for _, item := range report.Items {
		fmt.Fprintf(out, "%s expected=%d actual=%d\n", item.Name, item.Expected, item.Actual)
	}
}

func (report verifyReport) Validate() error {
	var missing []string
	for _, item := range report.Items {
		if item.Actual < item.Expected {
			missing = append(missing, fmt.Sprintf("%s expected=%d actual=%d", item.Name, item.Expected, item.Actual))
		}
	}
	if len(missing) > 0 {
		return errors.New(strings.Join(missing, "; "))
	}
	return nil
}

func uniqueRowIDs(rows []csvRow, key string) []int64 {
	seen := map[int64]struct{}{}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		id := rowInt(row, key)
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func contentIDsByType(rows []csvRow) ([]int64, []int64) {
	var articleIDs []int64
	var videoIDs []int64
	seenArticles := map[int64]struct{}{}
	seenVideos := map[int64]struct{}{}
	for _, row := range rows {
		contentID := rowInt(row, "content_id")
		if contentID <= 0 {
			continue
		}
		if contentTypeValue(row["scene"]) == 20 {
			if _, ok := seenVideos[contentID]; ok {
				continue
			}
			seenVideos[contentID] = struct{}{}
			videoIDs = append(videoIDs, contentID)
			continue
		}
		if _, ok := seenArticles[contentID]; ok {
			continue
		}
		seenArticles[contentID] = struct{}{}
		articleIDs = append(articleIDs, contentID)
	}
	return articleIDs, videoIDs
}

func uniqueFollowPairs(rows []csvRow) []followPair {
	seen := map[followPair]struct{}{}
	pairs := make([]followPair, 0, len(rows))
	for _, row := range rows {
		pair := followPair{
			FollowerID: rowInt(row, "follower_id"),
			FolloweeID: rowInt(row, "followee_id"),
		}
		if pair.FollowerID <= 0 || pair.FolloweeID <= 0 {
			continue
		}
		if _, ok := seen[pair]; ok {
			continue
		}
		seen[pair] = struct{}{}
		pairs = append(pairs, pair)
	}
	return pairs
}

func countRowsByIDs(ctx context.Context, db *sql.DB, table string, column string, ids []int64, extraWhere string) (int, error) {
	total := 0
	for _, batch := range chunkInt64s(ids, 500) {
		query := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE `%s` IN (%s)%s", table, column, placeholders(len(batch)), extraWhere)
		args := int64Args(batch)
		var count int
		if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func countRowsByFollowPairs(ctx context.Context, db *sql.DB, pairs []followPair) (int, error) {
	total := 0
	for _, batch := range chunkFollowPairs(pairs, 200) {
		clauses := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*2)
		for _, pair := range batch {
			clauses = append(clauses, "(`user_id` = ? AND `follow_user_id` = ?)")
			args = append(args, pair.FollowerID, pair.FolloweeID)
		}
		query := "SELECT COUNT(*) FROM `zfeed_follow` WHERE `status` = 10 AND `is_deleted` = 0 AND (" + strings.Join(clauses, " OR ") + ")"
		var count int
		if err := db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

func chunkInt64s(values []int64, size int) [][]int64 {
	if len(values) == 0 {
		return nil
	}
	var chunks [][]int64
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func chunkFollowPairs(values []followPair, size int) [][]followPair {
	if len(values) == 0 {
		return nil
	}
	var chunks [][]followPair
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func placeholders(count int) string {
	values := make([]string, count)
	for index := range values {
		values[index] = "?"
	}
	return strings.Join(values, ",")
}

func int64Args(values []int64) []any {
	args := make([]any, len(values))
	for index, value := range values {
		args[index] = value
	}
	return args
}

func rowInt(row csvRow, key string) int64 {
	value := strings.TrimSpace(row[key])
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func contentTypeValue(scene string) int {
	switch strings.ToUpper(strings.TrimSpace(scene)) {
	case "VIDEO":
		return 20
	default:
		return 10
	}
}

func sqlDateFromUnix(value string) string {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds <= 0 {
		return "NULL"
	}
	return sqlString(time.Unix(seconds, 0).UTC().Format("2006-01-02"))
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
