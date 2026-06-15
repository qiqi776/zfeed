package feedlogic

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	contentpb "zfeed/app/rpc/content/content"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

type fakeRecommendTrackProducer struct {
	events []track.Event
	err    error
}

func (p *fakeRecommendTrackProducer) Emit(ctx context.Context, event track.Event) error {
	p.events = append(p.events, event)
	return p.err
}

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

func newRecommendContentOnlyDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "_content_only?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedContent{}); err != nil {
		t.Fatalf("auto migrate content: %v", err)
	}
	return db
}

func TestRecommendHotFallbackRecordsReasonMetrics(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*miniredis.Miniredis, *svc.ServiceContext)
		wantErr      bool
		wantFallback string
	}{
		{
			name: "disabled enhancement records disabled fallback",
			setup: func(store *miniredis.Miniredis, svcCtx *svc.ServiceContext) {
				db := newFollowFeedTestDB(t)
				seedFollowFeedRows(t, db, []followFeedSeed{
					{contentID: 8801, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8801", coverURL: "cover-8801"},
				})
				store.ZAdd(redisconsts.HotFeedKey, 1000, "8801")
				svcCtx.MysqlDb = db
				svcCtx.Config.Recommend = contentconfig.RecommendConfig{
					Enabled: true,
					NewContent: contentconfig.RecommendNewContentConfig{
						Enabled: true,
						Weight:  1,
						Limit:   10,
					},
				}
				if err := svcCtx.Redis.HsetCtx(context.Background(), recommend.RuntimeFlagKey, "enabled", "false"); err != nil {
					t.Fatalf("hset runtime enabled=false: %v", err)
				}
			},
			wantFallback: recommendFallbackReasonDisabled,
		},
		{
			name:         "empty hot result records cold start fallback",
			setup:        func(store *miniredis.Miniredis, svcCtx *svc.ServiceContext) {},
			wantFallback: recommendFallbackReasonColdStart,
		},
		{
			name: "redis hot query error records hot error fallback",
			setup: func(store *miniredis.Miniredis, svcCtx *svc.ServiceContext) {
				store.Set(redisconsts.HotFeedKey, "not-a-zset")
			},
			wantErr:      true,
			wantFallback: recommendFallbackReasonHotError,
		},
		{
			name: "item build error records build error fallback",
			setup: func(store *miniredis.Miniredis, svcCtx *svc.ServiceContext) {
				db := newRecommendContentOnlyDB(t)
				publishedAt := time.Unix(8802, 0)
				if err := db.Create(&model.ZfeedContent{
					ID:          8802,
					UserID:      2001,
					ContentType: int32(contentpb.ContentType_ARTICLE),
					Status:      int32(contentpb.ContentStatus_PUBLISHED),
					Visibility:  int32(contentpb.Visibility_PUBLIC),
					PublishedAt: &publishedAt,
					IsDeleted:   0,
				}).Error; err != nil {
					t.Fatalf("create content row: %v", err)
				}
				store.ZAdd(redisconsts.HotFeedKey, 1000, "8802")
				svcCtx.MysqlDb = db
			},
			wantErr:      true,
			wantFallback: recommendFallbackReasonBuildError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, redisClient := newFollowFeedRedis(t)
			svcCtx := &svc.ServiceContext{Redis: redisClient}
			tt.setup(store, svcCtx)

			oldFallback := recordRecommendFallbackMetric
			defer func() {
				recordRecommendFallbackMetric = oldFallback
			}()
			fallbacks := map[string]int{}
			recordRecommendFallbackMetric = func(reason string) {
				fallbacks[reason]++
			}

			logic := NewRecommendFeedLogic(context.Background(), svcCtx)
			resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
				Cursor:   "",
				PageSize: 1,
			})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("RecommendFeed returned nil error, want error")
				}
			} else if err != nil {
				t.Fatalf("RecommendFeed returned error: %v", err)
			}
			if !tt.wantErr && resp == nil {
				t.Fatal("RecommendFeed returned nil response")
			}
			if fallbacks[tt.wantFallback] != 1 {
				t.Fatalf("fallback metrics = %#v, want %s", fallbacks, tt.wantFallback)
			}
		})
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

func TestRecommendEnhancementUsesCandidateCache(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8901, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8901", coverURL: "cover-8901"},
		{contentID: 9901, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9901", coverURL: "cover-9901"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9001, "8901")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9901")

	cfg := contentconfig.RecommendConfig{
		Enabled:      true,
		CandidateTTL: 120,
		Hot: contentconfig.RecommendHotConfig{
			Enabled: true,
			Weight:  1,
			Limit:   10,
		},
		NewContent: contentconfig.RecommendNewContentConfig{
			Enabled: true,
			Weight:  2,
			Limit:   10,
		},
		Diversity: contentconfig.RecommendDiversityConfig{
			Enabled: false,
		},
	}
	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config:  contentconfig.Config{Recommend: cfg},
		MysqlDb: db,
		Redis:   redisClient,
	})

	firstPage, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("first RecommendFeed returned error: %v", err)
	}
	if len(firstPage.GetItems()) == 0 {
		t.Fatal("first page is empty")
	}

	runtime := buildRecommendRuntime(cfg, 0)
	cacheKey := recommend.BuildCandidateCacheKey(0, runtime.variantID, runtime.configHash)
	if !store.Exists(cacheKey) {
		t.Fatalf("candidate cache key %q does not exist", cacheKey)
	}

	store.Del(redisconsts.HotFeedKey)
	store.Del(redisconsts.RecommendNewContentKey)

	secondPage, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("second RecommendFeed returned error: %v", err)
	}
	secondIDs := recommendContentIDs(secondPage.GetItems())
	if len(secondIDs) != 1 || secondIDs[0] != 9901 {
		t.Fatalf("second ids = %v, want cached top candidate [9901]", secondIDs)
	}
	if !strings.HasPrefix(secondPage.GetSnapshotId(), "rec:") {
		t.Fatalf("second snapshot_id = %q, want personalized snapshot", secondPage.GetSnapshotId())
	}
}

func TestRecommendEnhancementAppliesAndRecordsSeen(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8952, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8952", coverURL: "cover-8952"},
		{contentID: 9951, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9951", coverURL: "cover-9951"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9001, "8952")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9951")
	userID := int64(1001)
	seenKey := redisconsts.BuildRecommendSeenKey(userID)
	store.ZAdd(seenKey, float64(time.Now().Unix()), "9951")

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
					Weight:  2,
					Limit:   10,
				},
				Rank: contentconfig.RecommendRankConfig{
					AlphaHot:    1,
					GammaFresh:  1,
					SeenPenalty: 10,
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
		UserId:   &userID,
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	ids := recommendContentIDs(resp.GetItems())
	if len(ids) != 1 || ids[0] != 8952 {
		t.Fatalf("ids = %v, want previously unseen hot content [8952]", ids)
	}
	if _, err := store.ZScore(seenKey, "8952"); err != nil {
		t.Fatalf("returned content was not recorded in seen set: %v", err)
	}
}

func TestRecommendInterestRecallRecordsProfileMetrics(t *testing.T) {
	tests := []struct {
		name       string
		cfg        contentconfig.RecommendConfig
		setup      func(*miniredis.Miniredis)
		userID     int64
		wantResult string
	}{
		{
			name: "disabled",
			cfg: contentconfig.RecommendConfig{
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
				},
				Interest: contentconfig.RecommendInterestConfig{
					Enabled: false,
				},
			},
			userID:     1001,
			wantResult: recommendProfileResultDisabled,
		},
		{
			name: "skipped anonymous",
			cfg: contentconfig.RecommendConfig{
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
				},
				Interest: contentconfig.RecommendInterestConfig{
					Enabled: true,
				},
			},
			wantResult: recommendProfileResultSkipped,
		},
		{
			name: "miss",
			cfg: contentconfig.RecommendConfig{
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
				},
				Interest: contentconfig.RecommendInterestConfig{
					Enabled: true,
					MinTags: 1,
					TopTags: 1,
					Limit:   10,
				},
			},
			userID:     1001,
			wantResult: recommendProfileResultMiss,
		},
		{
			name: "hit",
			cfg: contentconfig.RecommendConfig{
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
				},
				Interest: contentconfig.RecommendInterestConfig{
					Enabled: true,
					MinTags: 1,
					TopTags: 1,
					Limit:   10,
				},
			},
			setup: func(store *miniredis.Miniredis) {
				store.HSet(redisconsts.BuildRecommendUserProfileKey(1001), "go", "1")
				store.ZAdd(redisconsts.BuildRecommendTagIndexKey("go"), 10, "7001")
			},
			userID:     1001,
			wantResult: recommendProfileResultHit,
		},
		{
			name: "error",
			cfg: contentconfig.RecommendConfig{
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
				},
				Interest: contentconfig.RecommendInterestConfig{
					Enabled: true,
					MinTags: 1,
					TopTags: 1,
					Limit:   10,
				},
			},
			setup: func(store *miniredis.Miniredis) {
				store.Set(redisconsts.BuildRecommendUserProfileKey(1001), "not-a-hash")
			},
			userID:     1001,
			wantResult: recommendProfileResultError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, redisClient := newFollowFeedRedis(t)
			if tt.setup != nil {
				tt.setup(store)
			}

			oldProfile := recordRecommendProfileMetric
			defer func() {
				recordRecommendProfileMetric = oldProfile
			}()

			results := map[string]int{}
			recordRecommendProfileMetric = func(result string) {
				results[result]++
			}

			logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
				Config: contentconfig.Config{
					Recommend: tt.cfg,
				},
				Redis: redisClient,
			})

			_, _, _ = logic.recallRecommendCandidates(
				&contentpb.RecommendFeedReq{UserId: &tt.userID},
				1,
				tt.cfg,
				recommendVariantControl,
			)

			if results[tt.wantResult] != 1 {
				t.Fatalf("profile metrics = %#v, want one %s", results, tt.wantResult)
			}
		})
	}
}

func TestRecallRecommendCandidatesTransfersInterestMissWeightToHot(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	store.ZAdd(redisconsts.HotFeedKey, 9001, "7001")

	userID := int64(1001)
	cfg := contentconfig.RecommendConfig{
		CandidateLimit: 10,
		Hot: contentconfig.RecommendHotConfig{
			Enabled: true,
			Weight:  0.55,
			Limit:   10,
		},
		Interest: contentconfig.RecommendInterestConfig{
			Enabled: true,
			Weight:  0.25,
			MinTags: 1,
			TopTags: 1,
			Limit:   10,
		},
	}
	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: cfg,
		},
		Redis: redisClient,
	})

	inputs, _, err := logic.recallRecommendCandidates(
		&contentpb.RecommendFeedReq{UserId: &userID},
		1,
		cfg,
		recommendVariantControl,
	)
	if err != nil {
		t.Fatalf("recallRecommendCandidates returned error: %v", err)
	}

	hotWeight := recallInputWeight(inputs, recommend.SourceHot)
	if hotWeight < 0.799 || hotWeight > 0.801 {
		t.Fatalf("hot weight = %v, want 0.80 after interest miss transfer; inputs=%#v", hotWeight, inputs)
	}
}

func TestRecallRecommendCandidatesPreservesInterestSourceScores(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	store.ZAdd(redisconsts.HotFeedKey, 9001, "7001")
	store.HSet(redisconsts.BuildRecommendUserProfileKey(1001), "go", "1")
	store.ZAdd(redisconsts.BuildRecommendTagIndexKey("go"), 100, "2001")
	store.ZAdd(redisconsts.BuildRecommendTagIndexKey("go"), 50, "2002")

	userID := int64(1001)
	cfg := contentconfig.RecommendConfig{
		CandidateLimit: 10,
		Hot: contentconfig.RecommendHotConfig{
			Enabled: true,
			Weight:  0.55,
			Limit:   10,
		},
		Interest: contentconfig.RecommendInterestConfig{
			Enabled: true,
			Weight:  0.25,
			MinTags: 1,
			TopTags: 1,
			Limit:   10,
		},
	}
	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: cfg,
		},
		Redis: redisClient,
	})

	inputs, _, err := logic.recallRecommendCandidates(
		&contentpb.RecommendFeedReq{UserId: &userID},
		1,
		cfg,
		recommendVariantControl,
	)
	if err != nil {
		t.Fatalf("recallRecommendCandidates returned error: %v", err)
	}

	merged := recommend.Merge(inputs, 10)
	if len(merged) != 3 {
		t.Fatalf("len(merged) = %d, want 3", len(merged))
	}
	if merged[0].ContentID != 7001 {
		t.Fatalf("first content id = %d, want hot 7001", merged[0].ContentID)
	}
	if merged[1].ContentID != 2001 || merged[2].ContentID != 2002 {
		t.Fatalf("interest ids = [%d %d], want [2001 2002]", merged[1].ContentID, merged[2].ContentID)
	}
	if merged[1].InterestScore != 1 {
		t.Fatalf("first interest score = %v, want 1", merged[1].InterestScore)
	}
	if merged[2].InterestScore != 0.5 {
		t.Fatalf("second interest score = %v, want 0.5", merged[2].InterestScore)
	}
	if merged[1].SourceRanks[recommend.SourceInterest] != 1 || merged[2].SourceRanks[recommend.SourceInterest] != 2 {
		t.Fatalf("source ranks = %#v/%#v, want 1/2", merged[1].SourceRanks, merged[2].SourceRanks)
	}
}

func TestRebalanceEmptyRecallWeightTransfersInterestMissToHot(t *testing.T) {
	tests := []struct {
		name          string
		inputs        []recommend.MergeInput
		wantHotWeight float64
	}{
		{
			name: "interest miss",
			inputs: []recommend.MergeInput{
				{
					Source: recommend.SourceHot,
					Weight: 0.55,
					IDs:    []int64{101, 102},
				},
				{
					Source: recommend.SourceInterest,
					Weight: 0.25,
					IDs:    nil,
				},
			},
			wantHotWeight: 0.80,
		},
		{
			name: "interest hit",
			inputs: []recommend.MergeInput{
				{
					Source: recommend.SourceHot,
					Weight: 0.55,
					IDs:    []int64{101, 102},
				},
				{
					Source: recommend.SourceInterest,
					Weight: 0.25,
					IDs:    []int64{201},
				},
			},
			wantHotWeight: 0.55,
		},
		{
			name: "hot miss",
			inputs: []recommend.MergeInput{
				{
					Source: recommend.SourceHot,
					Weight: 0.55,
					IDs:    nil,
				},
				{
					Source: recommend.SourceInterest,
					Weight: 0.25,
					IDs:    nil,
				},
			},
			wantHotWeight: 0.55,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rebalanceEmptyRecallWeight(tt.inputs, recommend.SourceInterest, recommend.SourceHot)

			if len(got) != len(tt.inputs) {
				t.Fatalf("rebalanceEmptyRecallWeight returned %d inputs, want %d", len(got), len(tt.inputs))
			}
			hotWeight := recallInputWeight(got, recommend.SourceHot)
			if hotWeight < tt.wantHotWeight-0.001 || hotWeight > tt.wantHotWeight+0.001 {
				t.Fatalf("hot weight = %v, want %v", hotWeight, tt.wantHotWeight)
			}
		})
	}
}

func recallInputWeight(inputs []recommend.MergeInput, source recommend.Source) float64 {
	for _, input := range inputs {
		if input.Source == source {
			return input.Weight
		}
	}
	return 0
}

func TestRecordRerankAdjustmentsPreservesRuleVariantAndCount(t *testing.T) {
	oldRerank := recordRecommendRerankAdjustMetric
	defer func() {
		recordRecommendRerankAdjustMetric = oldRerank
	}()

	records := []struct {
		rule    string
		variant string
		count   int
	}{}
	recordRecommendRerankAdjustMetric = func(rule, variant string, count int) {
		records = append(records, struct {
			rule    string
			variant string
			count   int
		}{rule: rule, variant: variant, count: count})
	}

	recordRerankAdjustments("b", map[string]int{
		recommend.DiversityRuleAuthorWindow: 2,
		recommend.DiversityRuleTypeWindow:   0,
	})

	if len(records) != 1 {
		t.Fatalf("records = %+v, want one positive adjustment", records)
	}
	if records[0].rule != recommend.DiversityRuleAuthorWindow ||
		records[0].variant != "b" ||
		records[0].count != 2 {
		t.Fatalf("record = %+v, want author_window/b/2", records[0])
	}
}

func TestRecommendEnhancementRecordsRecallErrorMetric(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	store.Set(redisconsts.RecommendNewContentKey, "not-a-zset")

	oldError := recordRecommendErrorMetric
	defer func() {
		recordRecommendErrorMetric = oldError
	}()
	records := []struct {
		stage   string
		variant string
	}{}
	recordRecommendErrorMetric = func(stage, variant string) {
		records = append(records, struct {
			stage   string
			variant string
		}{stage: stage, variant: variant})
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
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

	_, err := logic.recommendWithNewContent(
		&contentpb.RecommendFeedReq{Cursor: "", PageSize: 1},
		1,
		buildRecommendRuntime(logic.svcCtx.Config.Recommend, 0),
	)
	if err == nil {
		t.Fatal("recommendWithNewContent returned nil error, want recall error")
	}
	if len(records) != 1 {
		t.Fatalf("error records = %+v, want one recall error", records)
	}
	if records[0].stage != recommendStageRecall || records[0].variant != recommendVariantControl {
		t.Fatalf("error record = %+v, want recall/control", records[0])
	}
}

func TestRecommendFallbackToHotDisabledReturnsEnhancementError(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8461, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8461", coverURL: "cover-8461"},
	})
	store.ZAdd(redisconsts.HotFeedKey, 9001, "8461")
	store.Set(redisconsts.RecommendNewContentKey, "not-a-zset")

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled:       true,
				FallbackToHot: false,
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
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
	if err == nil {
		t.Fatalf("RecommendFeed returned nil error with resp=%+v, want enhancement error when fallback_to_hot is disabled", resp)
	}
	if resp != nil {
		t.Fatalf("RecommendFeed response = %+v, want nil on enhancement error", resp)
	}
}

func TestRecommendFallbackToHotEnabledKeepsHotFallbackOnEnhancementError(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8462, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8462", coverURL: "cover-8462"},
	})
	store.ZAdd(redisconsts.HotFeedKey, 9001, "8462")
	store.Set(redisconsts.RecommendNewContentKey, "not-a-zset")

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled:       true,
				FallbackToHot: true,
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
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
	if got := recommendContentIDs(resp.GetItems()); len(got) != 1 || got[0] != 8462 {
		t.Fatalf("ids = %v, want hot fallback [8462]", got)
	}
}

func TestRecommendEnhancementRecordsEmptyRecallFallback(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8451, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8451", coverURL: "cover-8451"},
	})
	store.ZAdd(redisconsts.HotFeedKey, 9001, "8451")

	oldFallback := recordRecommendFallbackMetric
	defer func() {
		recordRecommendFallbackMetric = oldFallback
	}()
	fallbacks := map[string]int{}
	recordRecommendFallbackMetric = func(reason string) {
		fallbacks[reason]++
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				Hot: contentconfig.RecommendHotConfig{
					Enabled: false,
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
	if got := recommendContentIDs(resp.GetItems()); len(got) != 1 || got[0] != 8451 {
		t.Fatalf("ids = %v, want hot fallback [8451]", got)
	}
	if fallbacks[recommendFallbackReasonEmptyRecall] != 1 {
		t.Fatalf("fallback metrics = %#v, want one empty_recall", fallbacks)
	}
}

func TestRecommendEnhancementEmitsExposureTrackEvents(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8961, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8961", coverURL: "cover-8961"},
		{contentID: 9961, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9961", coverURL: "cover-9961"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9001, "8961")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9961")

	producer := &fakeRecommendTrackProducer{}
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
					Weight:  2,
					Limit:   10,
				},
				Diversity: contentconfig.RecommendDiversityConfig{
					Enabled: false,
				},
			},
		},
		MysqlDb:                db,
		Redis:                  redisClient,
		RecommendTrackProducer: producer,
	})

	userID := int64(1001)
	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		UserId:   &userID,
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(resp.GetItems()))
	}
	if len(producer.events) != 1 {
		t.Fatalf("track events = %+v, want one exposure", producer.events)
	}
	event := producer.events[0]
	if event.EventType != track.EventTypeExposure {
		t.Fatalf("event type = %q, want exposure", event.EventType)
	}
	if event.UserID != userID || event.ContentID != resp.GetItems()[0].GetContentId() {
		t.Fatalf("event = %+v, want user/content from response", event)
	}
	if event.SnapshotID != resp.GetSnapshotId() || event.VariantID != recommendVariantControl {
		t.Fatalf("event snapshot/variant = %q/%q, want %q/control", event.SnapshotID, event.VariantID, resp.GetSnapshotId())
	}
	if event.Source != string(recommend.SourceNewContent) {
		t.Fatalf("event source = %q, want %q", event.Source, recommend.SourceNewContent)
	}
	if event.Position != 1 || event.OccurredAt <= 0 || event.EventID == "" {
		t.Fatalf("event metadata = %+v, want position/event_id/occurred_at", event)
	}
}

func TestRecommendExposureTrackEventsRecordEmitMetrics(t *testing.T) {
	tests := []struct {
		name       string
		emitErr    error
		wantResult string
	}{
		{
			name:       "success",
			wantResult: recommendResultSuccess,
		},
		{
			name:       "error",
			emitErr:    errors.New("kafka down"),
			wantResult: recommendResultError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldTrack := recordRecommendTrackEmitMetric
			defer func() {
				recordRecommendTrackEmitMetric = oldTrack
			}()

			records := []struct {
				eventType string
				result    string
			}{}
			recordRecommendTrackEmitMetric = func(eventType, result string) {
				records = append(records, struct {
					eventType string
					result    string
				}{eventType: eventType, result: result})
			}

			producer := &fakeRecommendTrackProducer{err: tt.emitErr}
			logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
				RecommendTrackProducer: producer,
			})

			logic.emitExposureTrackEvents(
				1001,
				&contentpb.RecommendFeedRes{
					SnapshotId: "rec:0001:control:hash:1",
					Items: []*contentpb.ContentItem{
						{ContentId: 9001},
						{ContentId: 9002},
					},
				},
				recommendVariantControl,
			)

			if len(records) != 2 {
				t.Fatalf("track metric records = %+v, want 2", records)
			}
			for _, record := range records {
				if record.eventType != track.EventTypeExposure || record.result != tt.wantResult {
					t.Fatalf("track metric record = %+v, want exposure/%s", record, tt.wantResult)
				}
			}
		})
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

func TestRecommendExperimentVariantWritesSnapshotMetaAndMetrics(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8501, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8501", coverURL: "cover-8501"},
		{contentID: 9501, authorID: 3001, contentType: contentpb.ContentType_VIDEO, title: "new-9501", coverURL: "cover-9501"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9001, "8501")
	store.ZAdd(redisconsts.RecommendNewContentKey, 9999, "9501")

	oldRequest := recordRecommendRequestMetric
	defer func() {
		recordRecommendRequestMetric = oldRequest
	}()
	requestVariants := []string{}
	recordRecommendRequestMetric = func(mode, variant, result string) {
		if mode == recommendModePersonalized && result == recommendResultSuccess {
			requestVariants = append(requestVariants, variant)
		}
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		Config: contentconfig.Config{
			Recommend: contentconfig.RecommendConfig{
				Enabled: true,
				NewContent: contentconfig.RecommendNewContentConfig{
					Enabled: true,
					Weight:  1,
					Limit:   10,
				},
				Experiment: contentconfig.RecommendExperimentConfig{
					ID:             "exp_rec_v1",
					Enabled:        true,
					Salt:           "salt",
					DefaultVariant: "control",
					Variants: []contentconfig.RecommendExperimentVariantConfig{
						{
							ID:               "b",
							TrafficPermyriad: 10000,
							Overrides: map[string]string{
								"rank.beta_interest": "0.40",
							},
						},
					},
				},
			},
		},
		MysqlDb: db,
		Redis:   redisClient,
	})

	userID := int64(1001)
	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		UserId:   &userID,
		Cursor:   "",
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	snapshotID := resp.GetSnapshotId()
	if !strings.Contains(snapshotID, ":b:") {
		t.Fatalf("snapshot_id = %q, want variant b embedded", snapshotID)
	}

	metaKey := redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID)
	if got := store.HGet(metaKey, "variant_id"); got != "b" {
		t.Fatalf("snapshot meta variant_id = %q, want b", got)
	}
	configHash := store.HGet(metaKey, "config_hash")
	if configHash == "" || configHash == "default" {
		t.Fatalf("snapshot meta config_hash = %q, want computed hash", configHash)
	}
	if len(requestVariants) != 1 || requestVariants[0] != "b" {
		t.Fatalf("request variants = %v, want [b]", requestVariants)
	}
}

func TestRecommendPersonalizedSnapshotRecordsHitMetric(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 8601, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "rec-8601", coverURL: "cover-8601"},
	})

	snapshotID := "rec:0001:control:hash8601:1"
	store.ZAdd(redisconsts.BuildRecommendUserSnapshotKey(snapshotID), 1000, "8601")

	oldRequest := recordRecommendRequestMetric
	oldStage := recordRecommendStageDurationMetric
	oldSnapshot := recordRecommendSnapshotMetric
	defer func() {
		recordRecommendRequestMetric = oldRequest
		recordRecommendStageDurationMetric = oldStage
		recordRecommendSnapshotMetric = oldSnapshot
	}()

	requests := []struct {
		mode   string
		result string
	}{}
	stages := map[string]int{}
	snapshots := map[string]int{}
	recordRecommendRequestMetric = func(mode, variant, result string) {
		requests = append(requests, struct {
			mode   string
			result string
		}{mode: mode, result: result})
	}
	recordRecommendStageDurationMetric = func(stage, variant string, elapsed time.Duration) {
		if elapsed < 0 {
			t.Fatalf("stage elapsed = %s, want non-negative", elapsed)
		}
		stages[stage]++
	}
	recordRecommendSnapshotMetric = func(kind, result string) {
		snapshots[kind+":"+result]++
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		PageSize:   1,
		SnapshotId: &snapshotID,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if got := recommendContentIDs(resp.GetItems()); len(got) != 1 || got[0] != 8601 {
		t.Fatalf("ids = %v, want [8601]", got)
	}
	if snapshots[recommendSnapshotKindPersonalized+":"+recommendSnapshotResultHit] != 1 {
		t.Fatalf("snapshot metrics = %#v, want personalized hit", snapshots)
	}
	if stages[recommendStageSnapshotLookup] != 1 {
		t.Fatalf("stage metrics = %#v, want snapshot lookup recorded", stages)
	}
	if len(requests) != 1 ||
		requests[0].mode != recommendModeSnapshot ||
		requests[0].result != recommendResultSuccess {
		t.Fatalf("request metrics = %+v, want one snapshot success", requests)
	}
}

func TestRecommendPersonalizedSnapshotRecordsMissAndErrorMetrics(t *testing.T) {
	tests := []struct {
		name         string
		prepare      func(*miniredis.Miniredis, string)
		wantSnapshot string
		wantFallback string
	}{
		{
			name: "missing snapshot falls back to hot path",
			prepare: func(store *miniredis.Miniredis, snapshotID string) {
				store.ZAdd(redisconsts.HotFeedKey, 1000, "8701")
			},
			wantSnapshot: recommendSnapshotResultMiss,
			wantFallback: recommendFallbackReasonSnapshotMiss,
		},
		{
			name: "redis error falls back to hot path",
			prepare: func(store *miniredis.Miniredis, snapshotID string) {
				store.Set(redisconsts.BuildRecommendUserSnapshotKey(snapshotID), "not-a-zset")
				store.ZAdd(redisconsts.HotFeedKey, 1000, "8701")
			},
			wantSnapshot: recommendSnapshotResultError,
			wantFallback: recommendFallbackReasonSnapshotError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, redisClient := newFollowFeedRedis(t)
			db := newFollowFeedTestDB(t)

			seedFollowFeedRows(t, db, []followFeedSeed{
				{contentID: 8701, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-8701", coverURL: "cover-8701"},
			})

			snapshotID := "rec:0001:control:hash8701:1"
			tt.prepare(store, snapshotID)

			oldRequest := recordRecommendRequestMetric
			oldStage := recordRecommendStageDurationMetric
			oldFallback := recordRecommendFallbackMetric
			oldSnapshot := recordRecommendSnapshotMetric
			defer func() {
				recordRecommendRequestMetric = oldRequest
				recordRecommendStageDurationMetric = oldStage
				recordRecommendFallbackMetric = oldFallback
				recordRecommendSnapshotMetric = oldSnapshot
			}()

			requests := []struct {
				mode   string
				result string
			}{}
			stages := map[string]int{}
			fallbacks := map[string]int{}
			snapshots := map[string]int{}
			recordRecommendRequestMetric = func(mode, variant, result string) {
				requests = append(requests, struct {
					mode   string
					result string
				}{mode: mode, result: result})
			}
			recordRecommendStageDurationMetric = func(stage, variant string, elapsed time.Duration) {
				if elapsed < 0 {
					t.Fatalf("stage elapsed = %s, want non-negative", elapsed)
				}
				stages[stage]++
			}
			recordRecommendFallbackMetric = func(reason string) {
				fallbacks[reason]++
			}
			recordRecommendSnapshotMetric = func(kind, result string) {
				snapshots[kind+":"+result]++
			}

			logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
				MysqlDb: db,
				Redis:   redisClient,
			})

			resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
				PageSize:   1,
				SnapshotId: &snapshotID,
			})
			if err != nil {
				t.Fatalf("RecommendFeed returned error: %v", err)
			}
			if got := recommendContentIDs(resp.GetItems()); len(got) != 1 || got[0] != 8701 {
				t.Fatalf("ids = %v, want hot fallback [8701]", got)
			}
			if snapshots[recommendSnapshotKindPersonalized+":"+tt.wantSnapshot] != 1 {
				t.Fatalf("snapshot metrics = %#v, want personalized %s", snapshots, tt.wantSnapshot)
			}
			if fallbacks[tt.wantFallback] != 1 {
				t.Fatalf("fallback metrics = %#v, want %s", fallbacks, tt.wantFallback)
			}
			if stages[recommendStageSnapshotLookup] != 1 {
				t.Fatalf("stage metrics = %#v, want snapshot lookup recorded", stages)
			}
			if len(requests) != 1 {
				t.Fatalf("request metrics = %+v, want one final request metric", requests)
			}
			lastRequest := requests[len(requests)-1]
			if lastRequest.mode != recommendModeHot || lastRequest.result != recommendResultSuccess {
				t.Fatalf("request metrics = %+v, want hot fallback success last", requests)
			}
		})
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

func TestRecommendPersonalizedSnapshotUsesSnapshotMetaVariant(t *testing.T) {
	store, redisClient := newFollowFeedRedis(t)
	db := newFollowFeedTestDB(t)

	seedFollowFeedRows(t, db, []followFeedSeed{
		{contentID: 6102, authorID: 2001, contentType: contentpb.ContentType_ARTICLE, title: "hot-6102", coverURL: "cover-6102"},
		{contentID: 6101, authorID: 2002, contentType: contentpb.ContentType_VIDEO, title: "hot-6101", coverURL: "cover-6101"},
	})

	store.ZAdd(redisconsts.HotFeedKey, 9002, "6102")
	store.ZAdd(redisconsts.HotFeedKey, 9001, "6101")

	snapshotID := "rec:0001:b:hash123:1"
	store.ZAdd(redisconsts.BuildRecommendUserSnapshotKey(snapshotID), 1000, "6102")
	store.HSet(redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID), "variant_id", "b")
	store.HSet(redisconsts.BuildRecommendUserSnapshotMetaKey(snapshotID), "config_hash", "hash123")

	oldRequest := recordRecommendRequestMetric
	defer func() {
		recordRecommendRequestMetric = oldRequest
	}()
	variantRequests := []string{}
	recordRecommendRequestMetric = func(mode, variant, result string) {
		if mode == recommendModeSnapshot && result == recommendResultSuccess {
			variantRequests = append(variantRequests, variant)
		}
	}

	logic := NewRecommendFeedLogic(context.Background(), &svc.ServiceContext{
		MysqlDb: db,
		Redis:   redisClient,
	})

	resp, err := logic.RecommendFeed(&contentpb.RecommendFeedReq{
		SnapshotId: &snapshotID,
		PageSize:   1,
	})
	if err != nil {
		t.Fatalf("RecommendFeed returned error: %v", err)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetContentId() != 6102 {
		t.Fatalf("items = %+v, want snapshot content 6102", recommendContentIDs(resp.GetItems()))
	}
	if len(variantRequests) != 1 || variantRequests[0] != "b" {
		t.Fatalf("variant requests = %v, want [b]", variantRequests)
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
