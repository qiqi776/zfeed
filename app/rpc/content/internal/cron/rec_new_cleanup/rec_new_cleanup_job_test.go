package rec_new_cleanup

import (
	"context"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/xxljob"
)

func TestRunRemovesExpiredNewContentRecall(t *testing.T) {
	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	now := time.Now().UTC()
	oldID := int64(1001)
	freshID := int64(1002)
	oldMember := strconv.FormatInt(oldID, 10)
	freshMember := strconv.FormatInt(freshID, 10)

	store.ZAdd(redisconsts.RecommendNewContentKey, float64(now.Add(-8*24*time.Hour).Unix()), oldMember)
	store.ZAdd(redisconsts.RecommendNewContentKey, float64(now.Add(-time.Hour).Unix()), freshMember)
	if err := client.HsetCtx(context.Background(), redisconsts.BuildRecommendNewContentMetaKey(oldID), "author_id", "7"); err != nil {
		t.Fatalf("seed old meta: %v", err)
	}
	if err := client.HsetCtx(context.Background(), redisconsts.BuildRecommendNewContentMetaKey(freshID), "author_id", "8"); err != nil {
		t.Fatalf("seed fresh meta: %v", err)
	}

	oldTagsKey := redisconsts.BuildRecommendContentTagsKey(oldID)
	if err := client.HsetCtx(context.Background(), oldTagsKey, "go", "0.8"); err != nil {
		t.Fatalf("seed old tags: %v", err)
	}
	tagKey := redisconsts.BuildRecommendTagIndexKey("go")
	store.ZAdd(tagKey, 100, oldMember)
	store.ZAdd(tagKey, 99, freshMember)

	job := &RecNewCleanupJob{
		svc: &svc.ServiceContext{Redis: client},
		now: func() time.Time { return now },
	}
	got, err := job.Run(context.Background(), xxljob.TriggerParam{
		ExecutorParams: `{"maxAgeHours":168,"batchSize":10}`,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "ok removed=1" {
		t.Fatalf("Run result = %q, want ok removed=1", got)
	}

	assertZSetMissing(t, store, redisconsts.RecommendNewContentKey, oldMember)
	assertZSetHas(t, store, redisconsts.RecommendNewContentKey, freshMember)
	if store.Exists(redisconsts.BuildRecommendNewContentMetaKey(oldID)) {
		t.Fatalf("old meta still exists")
	}
	if !store.Exists(redisconsts.BuildRecommendNewContentMetaKey(freshID)) {
		t.Fatalf("fresh meta was removed")
	}
	if store.Exists(oldTagsKey) {
		t.Fatalf("old content tags still exist")
	}
	assertZSetMissing(t, store, tagKey, oldMember)
	assertZSetHas(t, store, tagKey, freshMember)
}

func TestParseParams(t *testing.T) {
	got := parseParams(`{"maxAgeHours":72,"batchSize":50,"lockTtl":30}`)
	if got.MaxAgeHours != 72 || got.BatchSize != 50 || got.LockTTL != 30 {
		t.Fatalf("parseParams() = %+v", got)
	}
	if got := parseParams("{bad json"); got != (Params{}) {
		t.Fatalf("parseParams(invalid) = %+v, want zero", got)
	}
}

func assertZSetMissing(t *testing.T, store *miniredis.Miniredis, key string, member string) {
	t.Helper()

	members, err := store.ZMembers(key)
	if err != nil {
		t.Fatalf("zset %s members: %v", key, err)
	}
	for _, value := range members {
		if value == member {
			t.Fatalf("zset %s still has member %s: %v", key, member, members)
		}
	}
}

func assertZSetHas(t *testing.T, store *miniredis.Miniredis, key string, member string) {
	t.Helper()

	members, err := store.ZMembers(key)
	if err != nil {
		t.Fatalf("zset %s members: %v", key, err)
	}
	for _, value := range members {
		if value == member {
			return
		}
	}
	t.Fatalf("zset %s missing member %s: %v", key, member, members)
}
