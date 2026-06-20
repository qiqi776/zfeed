//go:build e2e

package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type recommendFeedReq struct {
	Cursor     string  `json:"cursor"`
	PageSize   uint32  `json:"page_size"`
	SnapshotID *string `json:"snapshot_id,omitempty"`
}

type recommendFeedItem struct {
	ContentID int64 `json:"content_id,string"`
}

type recommendFeedRes struct {
	Items      []recommendFeedItem `json:"items"`
	NextCursor string              `json:"next_cursor"`
	HasMore    bool                `json:"has_more"`
	SnapshotID string              `json:"snapshot_id"`
}

func TestRecommendHotSnapshotE2E(t *testing.T) {
	env := loadE2EEnv(t)
	client := newHTTPClient()
	db := openMySQL(t, env, true)
	redisClient := openRedis(t, env)

	contentIDs := loadHotContentIDs(t, db)
	if len(contentIDs) < 4 {
		t.Fatalf("need at least 4 published public contents, got %v", contentIDs)
	}

	snapshotID := fmt.Sprintf("recommend-demo-snapshot-%d", time.Now().UnixNano())
	snapshotKey := "feed:hot:global:snap:" + snapshotID
	latestKey := "feed:hot:global:latest"
	globalKey := "feed:hot:global"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	backups, err := backupRedisKeys(ctx, redisClient, latestKey, globalKey)
	if err != nil {
		t.Fatalf("backup redis keys: %v", err)
	}
	t.Cleanup(func() {
		if cleanupErr := redisClient.Del(ctx, snapshotKey).Err(); cleanupErr != nil {
			t.Fatalf("cleanup snapshot key: %v", cleanupErr)
		}
		if restoreErr := restoreRedisKeys(ctx, redisClient, backups); restoreErr != nil {
			t.Fatalf("restore redis keys: %v", restoreErr)
		}
	})

	if err := redisClient.Del(ctx, snapshotKey).Err(); err != nil {
		t.Fatalf("clear snapshot key: %v", err)
	}
	if err := redisClient.ZAdd(ctx, snapshotKey, redis.Z{Score: 3.8, Member: strconv.FormatInt(contentIDs[0], 10)},
		redis.Z{Score: 2.6, Member: strconv.FormatInt(contentIDs[1], 10)},
		redis.Z{Score: 1.9, Member: strconv.FormatInt(contentIDs[2], 10)},
		redis.Z{Score: 1.2, Member: strconv.FormatInt(contentIDs[3], 10)},
	).Err(); err != nil {
		t.Fatalf("seed snapshot zset: %v", err)
	}
	if err := redisClient.Set(ctx, latestKey, snapshotID, 0).Err(); err != nil {
		t.Fatalf("seed latest snapshot: %v", err)
	}

	firstStatus, firstBody := doJSONRequest(t, client, "POST", env.FrontAPIBaseURL+"/v1/feed/recommend", recommendFeedReq{
		Cursor:     "",
		PageSize:   2,
		SnapshotID: &snapshotID,
	}, "")
	firstRes := decodeJSONResponse[recommendFeedRes](t, firstStatus, firstBody)
	if got := recommendContentIDs(firstRes.Items); !equalInt64s(got, contentIDs[:2]) {
		t.Fatalf("first page ids = %v, want %v", got, contentIDs[:2])
	}
	if firstRes.NextCursor == "" {
		t.Fatal("first page next_cursor is empty")
	}

	secondStatus, secondBody := doJSONRequest(t, client, "POST", env.FrontAPIBaseURL+"/v1/feed/recommend", recommendFeedReq{
		Cursor:     firstRes.NextCursor,
		PageSize:   2,
		SnapshotID: &snapshotID,
	}, "")
	secondRes := decodeJSONResponse[recommendFeedRes](t, secondStatus, secondBody)
	if got := recommendContentIDs(secondRes.Items); !equalInt64s(got, contentIDs[2:4]) {
		t.Fatalf("second page ids = %v, want %v", got, contentIDs[2:4])
	}

	missingSnapshotID := "snapshot-not-exists"
	fallbackStatus, fallbackBody := doJSONRequest(t, client, "POST", env.FrontAPIBaseURL+"/v1/feed/recommend", recommendFeedReq{
		Cursor:     "",
		PageSize:   2,
		SnapshotID: &missingSnapshotID,
	}, "")
	fallbackRes := decodeJSONResponse[recommendFeedRes](t, fallbackStatus, fallbackBody)
	if fallbackRes.SnapshotID != snapshotID {
		t.Fatalf("fallback snapshot_id = %q, want %q", fallbackRes.SnapshotID, snapshotID)
	}

	if err := redisClient.Del(ctx, snapshotKey, latestKey, globalKey).Err(); err != nil {
		t.Fatalf("clear redis snapshot sources: %v", err)
	}
	missStatus, missBody := doJSONRequest(t, client, "POST", env.FrontAPIBaseURL+"/v1/feed/recommend", recommendFeedReq{
		Cursor:     "",
		PageSize:   2,
		SnapshotID: &missingSnapshotID,
	}, "")
	if missStatus < 400 && !strings.Contains(string(missBody), "热榜缓存不存在") {
		t.Fatalf("expected recommend error response to contain 热榜缓存不存在, got status=%d body=%s", missStatus, string(missBody))
	}
	if !strings.Contains(string(missBody), "热榜缓存不存在") {
		t.Fatalf("expected recommend error response to contain 热榜缓存不存在, got status=%d body=%s", missStatus, string(missBody))
	}
}

func loadHotContentIDs(t *testing.T, db *sql.DB) []int64 {
	t.Helper()

	if raw := strings.TrimSpace(os.Getenv("HOT_CONTENT_IDS")); raw != "" {
		fields := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\n' || r == '\t'
		})
		out := make([]int64, 0, len(fields))
		for _, field := range fields {
			value, err := strconv.ParseInt(field, 10, 64)
			if err != nil {
				t.Fatalf("parse HOT_CONTENT_IDS item %q: %v", field, err)
			}
			out = append(out, value)
		}
		return out
	}

	rows, err := db.Query(`
SELECT id
FROM zfeed_content
WHERE status = 30 AND visibility = 10 AND is_deleted = 0
ORDER BY id DESC
LIMIT 4;
`)
	if err != nil {
		t.Fatalf("query hot content ids: %v", err)
	}
	defer rows.Close()

	var out []int64
	for rows.Next() {
		var id int64
		if scanErr := rows.Scan(&id); scanErr != nil {
			t.Fatalf("scan hot content id: %v", scanErr)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate hot content ids: %v", err)
	}
	return out
}

func recommendContentIDs(items []recommendFeedItem) []int64 {
	out := make([]int64, 0, len(items))
	for _, item := range items {
		out = append(out, item.ContentID)
	}
	return out
}

func equalInt64s(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
