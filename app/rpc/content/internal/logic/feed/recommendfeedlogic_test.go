package feedlogic

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/svc"
)

func TestParseHit(t *testing.T) {
	res := []interface{}{
		int64(1),
		int64(1),
		"1002",
		"snap-20260410",
		"1010",
		"1002",
	}

	parsed, exists, err := parseHotFeedLuaResult(res)
	if err != nil {
		t.Fatalf("parseHotFeedLuaResult returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected exists=true, got false")
	}
	if parsed == nil {
		t.Fatalf("expected non-nil result")
	}
	if parsed.nextCursor != 1002 {
		t.Fatalf("expected nextCursor=1002, got %d", parsed.nextCursor)
	}
	if !parsed.hasMore {
		t.Fatalf("expected hasMore=true")
	}
	if parsed.resolvedSnapshotID != "snap-20260410" {
		t.Fatalf("expected resolvedSnapshotID=snap-20260410, got %s", parsed.resolvedSnapshotID)
	}
	if len(parsed.ids) != 2 || parsed.ids[0] != 1010 || parsed.ids[1] != 1002 {
		t.Fatalf("unexpected ids: %#v", parsed.ids)
	}
}

func TestParseMiss(t *testing.T) {
	res := []interface{}{
		int64(0),
		int64(0),
		"",
		"",
	}

	parsed, exists, err := parseHotFeedLuaResult(res)
	if err != nil {
		t.Fatalf("parseHotFeedLuaResult returned error: %v", err)
	}
	if exists {
		t.Fatalf("expected exists=false, got true")
	}
	if parsed == nil {
		t.Fatalf("expected non-nil result")
	}
	if parsed.hasMore {
		t.Fatalf("expected hasMore=false")
	}
	if parsed.nextCursor != 0 {
		t.Fatalf("expected nextCursor=0, got %d", parsed.nextCursor)
	}
	if len(parsed.ids) != 0 {
		t.Fatalf("expected empty ids, got %#v", parsed.ids)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, _, err := parseHotFeedLuaResult([]interface{}{int64(1)}); err == nil {
		t.Fatalf("expected error for invalid lua result shape")
	}
}

func TestParseCursor(t *testing.T) {
	res := []interface{}{
		int64(1),
		int64(1),
		"abc",
		"snap-x",
		"1001",
	}
	if _, _, err := parseHotFeedLuaResult(res); err == nil {
		t.Fatalf("expected error for invalid next cursor")
	}
}

func TestMapCache(t *testing.T) {
	cases := []struct {
		name string
		in   CacheResult
		want string
	}{
		{
			name: "cache error",
			in:   cacheError,
			want: "查询热榜索引失败",
		},
		{
			name: "unknown cache result",
			in:   CacheResult(999),
			want: "查询失败请稍后重试",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapHotFeedCacheError(tc.in)
			if err == nil {
				t.Fatalf("expected non-nil error")
			}
			if err.Error() != tc.want {
				t.Fatalf("unexpected error message, got=%q want=%q", err.Error(), tc.want)
			}
		})
	}
}

func TestColdStart(t *testing.T) {
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Redis: redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 0 || resp.GetHasMore() || resp.GetNextCursor() != 0 || resp.GetSnapshotId() != "" {
		t.Fatalf("unexpected cold start response: %+v", resp)
	}
}

func TestQueryHotIDsCursorZeroUsesFirstPage(t *testing.T) {
	store := miniredis.RunT(t)
	store.ZAdd(redisconsts.HotFeedKey, 9003, "1003")
	store.ZAdd(redisconsts.HotFeedKey, 9002, "1002")
	store.ZAdd(redisconsts.HotFeedKey, 9001, "1001")

	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Redis: redisClient,
	})

	result, cacheResult := logic.queryFromRedis("", "", "0", 2)
	if cacheResult != cacheHit {
		t.Fatalf("expected cache hit, got %v", cacheResult)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
	if len(result.ids) != 2 || result.ids[0] != 1003 || result.ids[1] != 1002 {
		t.Fatalf("unexpected ids: %#v", result.ids)
	}
	if !result.hasMore {
		t.Fatalf("expected hasMore=true")
	}
	if result.nextCursor != 1002 {
		t.Fatalf("expected nextCursor=1002, got %d", result.nextCursor)
	}
}

func TestRecommendEnhancementMixesNewContent(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 4003, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-4003", coverURL: "cover-4003"},
		{contentID: 4002, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-4002", coverURL: "cover-4002"},
		{contentID: 4001, authorID: 2002, contentType: contentpb.ContentType_VIDEO, title: "hot-4001", coverURL: "cover-4001"},
		{contentID: 5001, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-5001", coverURL: "cover-5001"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9003, "4003")
	store.ZAdd(redisconsts.HotFeedKey, 9002, "4002")
	store.ZAdd(redisconsts.HotFeedKey, 9001, "4001")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "5001")

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled:        true,
				CandidateLimit: 10,
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  2,
					Limit:   10,
				},
				Diversity: contentconfig.RecommendDiversityConfig{
					Enabled:       true,
					AuthorWindow:  2,
					MaxSameAuthor: 1,
					TypeWindow:    3,
					MaxSameType:   2,
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(resp.GetItems()))
	}

	ids := []int64{
		resp.GetItems()[0].GetContentId(),
		resp.GetItems()[1].GetContentId(),
		resp.GetItems()[2].GetContentId(),
	}
	if !containsInt64(ids, 5001) {
		t.Fatalf("items = %v, want mixed new content 5001", ids)
	}
	if ids[0] != 5001 {
		t.Fatalf("first item = %d, want boosted new content 5001", ids[0])
	}
	if !strings.HasPrefix(resp.GetSnapshotId(), "rec:") {
		t.Fatalf("snapshot_id = %q, want personalized rec snapshot", resp.GetSnapshotId())
	}
}

func TestRecommendRuntimeFlagDisablesEnhancement(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8103, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8103", coverURL: "cover-8103"},
		{contentID: 8102, authorID: 2002, contentType: contentpb.ContentType_VIDEO, title: "hot-8102", coverURL: "cover-8102"},
		{contentID: 8101, authorID: 2003, contentType: contentpb.ContentType_ARTICLE, title: "hot-8101", coverURL: "cover-8101"},
		{contentID: 9101, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9101", coverURL: "cover-9101"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9003, "8103")
	store.ZAdd(redisconsts.HotFeedKey, 9002, "8102")
	store.ZAdd(redisconsts.HotFeedKey, 9001, "8101")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9101")
	if err := redisClient.HsetCtx(context.Background(), recommend.RuntimeFlagKey, "enabled", "false"); err != nil {
		t.Fatalf("hset runtime enabled=false: %v", err)
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  10,
					Limit:   10,
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 3,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}

	ids := recommendContentIDs(resp.GetItems())
	wantIDs := []int64{8103, 8102, 8101}
	if len(ids) != len(wantIDs) {
		t.Fatalf("ids = %v, want %v", ids, wantIDs)
	}
	for i := range wantIDs {
		if ids[i] != wantIDs[i] {
			t.Fatalf("ids = %v, want %v", ids, wantIDs)
		}
	}
	if strings.HasPrefix(resp.GetSnapshotId(), "rec:") {
		t.Fatalf("snapshot_id = %q, want hot fallback snapshot", resp.GetSnapshotId())
	}
}

func TestRecommendFineRankUsesConfiguredWeightsInMainPath(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8202, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8202", coverURL: "cover-8202"},
		{contentID: 8201, authorID: 2002, contentType: contentpb.ContentType_ARTICLE, title: "hot-8201", coverURL: "cover-8201"},
		{contentID: 9201, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9201", coverURL: "cover-9201"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9002, "8202")
	store.ZAdd(redisconsts.HotFeedKey, 9001, "8201")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9201")

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  1,
					Limit:   10,
				},
				Rank: contentconfig.RecommendRankConfig{
					AlphaHot: 1,
				},
				Diversity: contentconfig.RecommendDiversityConfig{
					Enabled: false,
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}

	ids := recommendContentIDs(resp.GetItems())
	if len(ids) < 2 {
		t.Fatalf("ids = %v, want at least 2 items", ids)
	}
	if ids[0] != 8202 {
		t.Fatalf("first id = %d, want hot candidate 8202 after fine rank: %v", ids[0], ids)
	}
}

func TestRecommendRuntimeFlagDisablesHotRecallInEnhancedPath(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8301, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8301", coverURL: "cover-8301"},
		{contentID: 9301, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9301", coverURL: "cover-9301"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9001, "8301")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9301")
	if err := redisClient.HsetCtx(context.Background(), recommend.RuntimeFlagKey, "recall.hot.enabled", "false"); err != nil {
		t.Fatalf("hset hot enabled=false: %v", err)
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				Hot: contentconfig.RecommendHotConfig{
					Enabled: true,
					Weight:  1,
					Limit:   10,
				},
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  0.1,
					Limit:   10,
				},
				Diversity: contentconfig.RecommendDiversityConfig{
					Enabled: false,
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}

	ids := recommendContentIDs(resp.GetItems())
	if len(ids) != 1 || ids[0] != 9301 {
		t.Fatalf("ids = %v, want only new content 9301 when hot recall disabled", ids)
	}
	if !strings.HasPrefix(resp.GetSnapshotId(), "rec:") {
		t.Fatalf("snapshot_id = %q, want personalized rec snapshot", resp.GetSnapshotId())
	}
}

func TestRecommendWithTimeoutUsesConfiguredBudget(t *testing.T) {
	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{})

	scoped, cancel := logic.withRecommendTimeout(contentconfig.RecommendConfig{TimeoutMs: 250})
	defer cancel()

	if scoped == logic {
		t.Fatal("withRecommendTimeout returned original logic, want scoped clone")
	}
	deadline, ok := scoped.ctx.Deadline()
	if !ok {
		t.Fatal("scoped context has no deadline")
	}
	remaining := time.Until(deadline)
	if remaining <= 0 || remaining > 250*time.Millisecond {
		t.Fatalf("deadline remaining = %s, want within 250ms", remaining)
	}
	if scoped.itemBuilder == logic.itemBuilder {
		t.Fatal("itemBuilder was not rebuilt with scoped context")
	}
}

func TestRecommendEnhancementRecordsCoreMetrics(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8401, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8401", coverURL: "cover-8401"},
		{contentID: 9401, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9401", coverURL: "cover-9401"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9001, "8401")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9401")

	oldRequest := recordRecommendRequestMetric
	oldStage := recordRecommendStageDurationMetric
	oldRecall := recordRecommendRecallItemsMetric
	oldSnapshot := recordRecommendSnapshotMetric
	defer func() {
		recordRecommendRequestMetric = oldRequest
		recordRecommendStageDurationMetric = oldStage
		recordRecommendRecallItemsMetric = oldRecall
		recordRecommendSnapshotMetric = oldSnapshot
	}()

	requests := []struct {
		mode    string
		variant string
		result  string
	}{}
	stages := map[string]int{}
	recalls := map[string]int{}
	snapshots := map[string]int{}
	recordRecommendRequestMetric = func(mode, variant, result string) {
		requests = append(requests, struct {
			mode    string
			variant string
			result  string
		}{mode: mode, variant: variant, result: result})
	}
	recordRecommendStageDurationMetric = func(stage, variant string, elapsed time.Duration) {
		if variant != recommendVariantControl {
			t.Fatalf("stage variant = %q, want control", variant)
		}
		if elapsed < 0 {
			t.Fatalf("stage elapsed = %s, want non-negative", elapsed)
		}
		stages[stage]++
	}
	recordRecommendRecallItemsMetric = func(source, variant string, count int) {
		recalls[source] += count
	}
	recordRecommendSnapshotMetric = func(kind, result string) {
		snapshots[kind+":"+result]++
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				Hot: contentconfig.RecommendHotConfig{
					Enabled: true,
					Weight:  1,
					Limit:   10,
				},
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  1,
					Limit:   10,
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if len(resp.GetItems()) == 0 {
		t.Fatal("expected non-empty response")
	}
	if len(requests) != 1 ||
		requests[0].mode != recommendModePersonalized ||
		requests[0].variant != recommendVariantControl ||
		requests[0].result != recommendResultSuccess {
		t.Fatalf("request metrics = %+v, want one personalized success", requests)
	}
	if recalls[recommendRecallSourceHot] != 1 {
		t.Fatalf("hot recall metric = %d, want 1", recalls[recommendRecallSourceHot])
	}
	if recalls[recommendRecallSourceNewContent] != 1 {
		t.Fatalf("new content recall metric = %d, want 1", recalls[recommendRecallSourceNewContent])
	}
	if snapshots[recommendSnapshotKindPersonalized+":"+recommendSnapshotResultSaved] != 1 {
		t.Fatalf("snapshot metrics = %#v, want personalized saved", snapshots)
	}
	wantStages := []string{
		recommendStageRecall,
		recommendStageCoarseRank,
		recommendStageFeatureLoad,
		recommendStageFineRank,
		recommendStageRerank,
		recommendStageSnapshotSave,
	}
	for _, stage := range wantStages {
		if stages[stage] == 0 {
			t.Fatalf("stages = %#v, want %s recorded", stages, stage)
		}
	}
}

func TestRecommendEnhancementUsesPersonalizedSnapshotForNextPage(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 6004, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-6004", coverURL: "cover-6004"},
		{contentID: 6003, authorID: 2002, contentType: contentpb.ContentType_VIDEO, title: "hot-6003", coverURL: "cover-6003"},
		{contentID: 6002, authorID: 2003, contentType: contentpb.ContentType_ARTICLE, title: "hot-6002", coverURL: "cover-6002"},
		{contentID: 6001, authorID: 2004, contentType: contentpb.ContentType_VIDEO, title: "hot-6001", coverURL: "cover-6001"},
		{contentID: 7001, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-7001", coverURL: "cover-7001"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9004, "6004")
	store.ZAdd(redisconsts.HotFeedKey, 9003, "6003")
	store.ZAdd(redisconsts.HotFeedKey, 9002, "6002")
	store.ZAdd(redisconsts.HotFeedKey, 9001, "6001")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "7001")

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  10,
					Limit:   10,
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	firstPage, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if len(firstPage.GetItems()) != 2 {
		t.Fatalf("len(firstPage.items) = %d, want 2", len(firstPage.GetItems()))
	}
	if !strings.HasPrefix(firstPage.GetSnapshotId(), "rec:") {
		t.Fatalf("snapshot_id = %q, want personalized rec snapshot", firstPage.GetSnapshotId())
	}
	if !firstPage.GetHasMore() {
		t.Fatalf("firstPage.has_more = false, want true")
	}

	snapshotID := firstPage.GetSnapshotId()
	secondPage, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:     strconv.FormatInt(firstPage.GetNextCursor(), 10),
		PageSize:   2,
		SnapshotId: &snapshotID,
	})
	if err != nil {
		t.Fatalf("second page returned error: %v", err)
	}
	if secondPage.GetSnapshotId() != firstPage.GetSnapshotId() {
		t.Fatalf("second snapshot_id = %q, want %q", secondPage.GetSnapshotId(), firstPage.GetSnapshotId())
	}
	gotSecondIDs := []int64{}
	for _, item := range secondPage.GetItems() {
		gotSecondIDs = append(gotSecondIDs, item.GetContentId())
	}
	wantSecondIDs := []int64{6003, 6002}
	if len(gotSecondIDs) != len(wantSecondIDs) {
		t.Fatalf("second ids = %v, want %v", gotSecondIDs, wantSecondIDs)
	}
	for i := range wantSecondIDs {
		if gotSecondIDs[i] != wantSecondIDs[i] {
			t.Fatalf("second ids = %v, want %v", gotSecondIDs, wantSecondIDs)
		}
	}
}

func recommendContentIDs(items []*contentpb.ContentItem) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		ids = append(ids, item.GetContentId())
	}
	return ids
}

func containsInt64(values []int64, want int64) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
