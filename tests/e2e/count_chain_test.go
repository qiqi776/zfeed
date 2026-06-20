//go:build e2e

package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

const countConsumerName = "count.canal_consumer"

type countValueRow struct {
	BizType    int32
	TargetType int32
	TargetID   int64
	OwnerID    int64
	Value      int64
	Version    int64
}

func TestCountChainE2E(t *testing.T) {
	env := loadE2EEnv(t)
	db := openMySQL(t, env, true)
	redisClient := openRedis(t, env)

	writer := &kafka.Writer{
		Addr:     kafka.TCP(env.KafkaBrokers...),
		Topic:    "zfeed-count-canal",
		Balancer: &kafka.LeastBytes{},
	}
	t.Cleanup(func() { _ = writer.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	baseID := time.Now().UnixNano() / int64(time.Millisecond)
	likeTargetID := baseID
	followingTargetID := baseID + 1
	followedTargetID := baseID + 2
	ownerID := followedTargetID
	eventPrefix := fmt.Sprintf("count-e2e-%d", baseID)

	countValueKey := fmt.Sprintf("count:value:10:10:%d", likeTargetID)
	ownerProfileKey := fmt.Sprintf("count:user:profile:%d", ownerID)
	followingProfileKey := fmt.Sprintf("count:user:profile:%d", followingTargetID)

	cleanup := func() {
		if err := cleanupCountChainState(ctx, db, redisClient, eventPrefix, likeTargetID, followingTargetID, followedTargetID, countValueKey, ownerProfileKey, followingProfileKey); err != nil {
			t.Fatalf("cleanup count chain state: %v", err)
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	if err := redisClient.Set(ctx, countValueKey, "sentinel-like", 0).Err(); err != nil {
		t.Fatalf("seed count cache: %v", err)
	}
	if err := redisClient.Set(ctx, ownerProfileKey, "sentinel-owner", 0).Err(); err != nil {
		t.Fatalf("seed owner profile cache: %v", err)
	}
	if err := redisClient.Set(ctx, followingProfileKey, "sentinel-following", 0).Err(); err != nil {
		t.Fatalf("seed following profile cache: %v", err)
	}

	insertMessages := []kafka.Message{
		{Value: []byte(fmt.Sprintf(`{"id":"%s-like-insert","table":"zfeed_like","type":"INSERT","ts":1775553600000,"data":[{"id":900021,"user_id":12001,"content_id":%d,"content_user_id":%d,"status":10,"is_deleted":0}],"old":[]}`, eventPrefix, likeTargetID, ownerID))},
		{Value: []byte(fmt.Sprintf(`{"id":"%s-follow-insert","table":"zfeed_follow","type":"INSERT","ts":1775553601000,"data":[{"id":900022,"user_id":%d,"follow_user_id":%d,"status":10,"is_deleted":0}],"old":[]}`, eventPrefix, followingTargetID, followedTargetID))},
	}
	if err := writer.WriteMessages(ctx, insertMessages...); err != nil {
		t.Fatalf("write insert messages: %v", err)
	}

	requireEventually(t, 30*time.Second, 500*time.Millisecond, func() error {
		rows, err := queryCountRows(ctx, db, likeTargetID, followingTargetID, followedTargetID)
		if err != nil {
			return err
		}
		if err := requireCountRow(rows, 10, 10, likeTargetID, ownerID, 1, 1); err != nil {
			return err
		}
		if err := requireCountRow(rows, 41, 20, followingTargetID, 0, 1, 1); err != nil {
			return err
		}
		if err := requireCountRow(rows, 40, 20, followedTargetID, 0, 1, 1); err != nil {
			return err
		}
		return nil
	})

	updateMessages := []kafka.Message{
		{Value: []byte(fmt.Sprintf(`{"id":"%s-like-update","table":"zfeed_like","type":"UPDATE","ts":1775553660000,"data":[{"id":900021,"user_id":12001,"content_id":%d,"content_user_id":%d,"status":20,"is_deleted":0}],"old":[{"status":10}]}`, eventPrefix, likeTargetID, ownerID))},
		{Value: []byte(fmt.Sprintf(`{"id":"%s-follow-update","table":"zfeed_follow","type":"UPDATE","ts":1775553661000,"data":[{"id":900022,"user_id":%d,"follow_user_id":%d,"status":20,"is_deleted":0}],"old":[{"status":10,"is_deleted":0}]}`, eventPrefix, followingTargetID, followedTargetID))},
	}
	if err := writer.WriteMessages(ctx, updateMessages...); err != nil {
		t.Fatalf("write update messages: %v", err)
	}

	requireEventually(t, 30*time.Second, 500*time.Millisecond, func() error {
		rows, err := queryCountRows(ctx, db, likeTargetID, followingTargetID, followedTargetID)
		if err != nil {
			return err
		}
		if err := requireCountRow(rows, 10, 10, likeTargetID, ownerID, 0, 2); err != nil {
			return err
		}
		if err := requireCountRow(rows, 41, 20, followingTargetID, 0, 0, 2); err != nil {
			return err
		}
		if err := requireCountRow(rows, 40, 20, followedTargetID, 0, 0, 2); err != nil {
			return err
		}

		for _, key := range []string{countValueKey, ownerProfileKey, followingProfileKey} {
			value, redisErr := redisGetString(ctx, redisClient, key)
			if redisErr != nil {
				return fmt.Errorf("read redis key %s: %w", key, redisErr)
			}
			if value != "" {
				return fmt.Errorf("expected redis key %s to be deleted, got %q", key, value)
			}
		}
		return nil
	})
}

func cleanupCountChainState(ctx context.Context, db *sql.DB, redisClient *redis.Client, eventPrefix string, likeTargetID, followingTargetID, followedTargetID int64, cacheKeys ...string) error {
	if _, err := db.ExecContext(ctx, `
DELETE FROM zfeed_count_value
WHERE (biz_type, target_type, target_id) IN ((10, 10, ?), (41, 20, ?), (40, 20, ?));
`, likeTargetID, followingTargetID, followedTargetID); err != nil {
		return fmt.Errorf("reset count chain rows: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
DELETE FROM zfeed_mq_consume_dedup
WHERE consumer = ? AND event_id LIKE ?;
`, countConsumerName, eventPrefix+"%"); err != nil {
		return fmt.Errorf("reset count dedup rows: %w", err)
	}
	if err := redisClient.Del(ctx, cacheKeys...).Err(); err != nil {
		return fmt.Errorf("reset count chain redis keys: %w", err)
	}
	return nil
}

func queryCountRows(ctx context.Context, db *sql.DB, likeTargetID, followingTargetID, followedTargetID int64) (map[string]countValueRow, error) {
	rows, err := db.QueryContext(ctx, `
SELECT biz_type, target_type, target_id, owner_id, value, version
FROM zfeed_count_value
WHERE (biz_type, target_type, target_id) IN ((10, 10, ?), (41, 20, ?), (40, 20, ?));
`, likeTargetID, followingTargetID, followedTargetID)
	if err != nil {
		return nil, fmt.Errorf("query count rows: %w", err)
	}
	defer rows.Close()

	out := make(map[string]countValueRow, 3)
	for rows.Next() {
		var row countValueRow
		if scanErr := rows.Scan(&row.BizType, &row.TargetType, &row.TargetID, &row.OwnerID, &row.Value, &row.Version); scanErr != nil {
			return nil, fmt.Errorf("scan count row: %w", scanErr)
		}
		out[countRowKey(row.BizType, row.TargetType, row.TargetID)] = row
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate count rows: %w", err)
	}
	return out, nil
}

func requireCountRow(rows map[string]countValueRow, bizType, targetType int32, targetID, ownerID, value, version int64) error {
	row, ok := rows[countRowKey(bizType, targetType, targetID)]
	if !ok {
		return fmt.Errorf("missing count row %s", countRowKey(bizType, targetType, targetID))
	}
	if row.OwnerID != ownerID || row.Value != value || row.Version != version {
		return fmt.Errorf("unexpected row %s: owner=%d value=%d version=%d", countRowKey(bizType, targetType, targetID), row.OwnerID, row.Value, row.Version)
	}
	return nil
}

func countRowKey(bizType, targetType int32, targetID int64) string {
	return strings.Join([]string{
		strconv.FormatInt(int64(bizType), 10),
		strconv.FormatInt(int64(targetType), 10),
		strconv.FormatInt(targetID, 10),
	}, ":")
}
