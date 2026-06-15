package rec_tag_refresh

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	"gorm.io/gorm"
	redisconsts "zfeed/app/rpc/content/internal/common/consts/redis"
	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/do"
	"zfeed/app/rpc/content/internal/model"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/repositories"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/xxljob"
)

type fakeTagRefreshContentRepo struct {
	rows []*model.ZfeedContent
}

func (r *fakeTagRefreshContentRepo) WithTx(_ *gorm.DB) repositories.ContentRepository {
	panic("WithTx is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) CreateContent(_ *do.ContentDO) (int64, error) {
	panic("CreateContent is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) ListLatestPublishedIDsByAuthor(_ int64, _ int) ([]int64, error) {
	panic("ListLatestPublishedIDsByAuthor is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) BatchGetAuthorsByIDs(_ []int64) (map[int64]int64, error) {
	panic("BatchGetAuthorsByIDs is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) BatchGetPublishedByIDs(_ []int64) (map[int64]*model.ZfeedContent, error) {
	panic("BatchGetPublishedByIDs is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) ListFollowByAuthorsCursor(_ []int64, _ int64, _ int) ([]*model.ZfeedContent, error) {
	panic("ListFollowByAuthorsCursor is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) ListPublishedByAuthor(_ int64) ([]*model.ZfeedContent, error) {
	panic("ListPublishedByAuthor is not used by rec tag refresh")
}

func (r *fakeTagRefreshContentRepo) ListColdUpdateContents(
	_ int32,
	_ int32,
	_ time.Time,
	cursorID int64,
	limit int,
) ([]*model.ZfeedContent, error) {
	out := make([]*model.ZfeedContent, 0, limit)
	for _, row := range r.rows {
		if row == nil || (cursorID > 0 && row.ID >= cursorID) {
			continue
		}
		out = append(out, row)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (r *fakeTagRefreshContentRepo) BatchUpdateHotScores(_ []int64, _ []float64, _ time.Time) error {
	panic("BatchUpdateHotScores is not used by rec tag refresh")
}

func TestRunRefreshesRecommendationTagIndexes(t *testing.T) {
	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-2 * time.Hour)
	rows := []*model.ZfeedContent{
		{
			ID:            2002,
			LikeCount:     8,
			CommentCount:  1,
			FavoriteCount: 1,
			PublishedAt:   &publishedAt,
		},
		{
			ID:           2001,
			LikeCount:    4,
			PublishedAt:  &publishedAt,
			CommentCount: 0,
		},
	}

	if err := recommend.WriteContentTags(
		context.Background(),
		client,
		contentconfig.RecommendConfig{},
		2001,
		map[string]float64{"go": 0.5},
		1,
	); err != nil {
		t.Fatalf("seed content 2001 tags: %v", err)
	}
	if err := recommend.WriteContentTags(
		context.Background(),
		client,
		contentconfig.RecommendConfig{},
		2002,
		map[string]float64{"go": 1},
		1,
	); err != nil {
		t.Fatalf("seed content 2002 tags: %v", err)
	}

	job := &RecTagRefreshJob{
		svc: &svc.ServiceContext{
			Config: contentconfig.Config{
				Recommend: contentconfig.RecommendConfig{
					Interest: contentconfig.RecommendInterestConfig{
						ContentTagTTL: 300,
						TagIndexTTL:   120,
					},
				},
			},
			Redis: client,
		},
		contentRepo: &fakeTagRefreshContentRepo{rows: rows},
		now:         func() time.Time { return now },
	}

	got, err := job.Run(context.Background(), xxljob.TriggerParam{
		ExecutorParams: `{"windowDays":7,"pageSize":1,"batchSize":1,"lockTtl":30}`,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "ok refreshed=2" {
		t.Fatalf("Run result = %q, want ok refreshed=2", got)
	}

	tagKey := redisconsts.BuildRecommendTagIndexKey("go")
	score2002, err := store.ZScore(tagKey, "2002")
	if err != nil {
		t.Fatalf("tag index score 2002: %v", err)
	}
	score2001, err := store.ZScore(tagKey, "2001")
	if err != nil {
		t.Fatalf("tag index score 2001: %v", err)
	}
	if score2002 <= score2001 {
		t.Fatalf("tag index scores 2002=%v 2001=%v, want 2002 higher", score2002, score2001)
	}
	if ttl := store.TTL(tagKey); ttl <= 0 || ttl > 120*time.Second {
		t.Fatalf("tag index ttl = %s, want within 120s", ttl)
	}
}

func TestRunRefreshesTagIndexForFreshContentWithoutInteractions(t *testing.T) {
	store := miniredis.RunT(t)
	client := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	publishedAt := now.Add(-30 * time.Minute)
	contentID := int64(2001)
	member := "2001"

	if err := recommend.WriteContentTags(
		context.Background(),
		client,
		contentconfig.RecommendConfig{},
		contentID,
		map[string]float64{"go": 1},
		999,
	); err != nil {
		t.Fatalf("seed content tags: %v", err)
	}

	job := &RecTagRefreshJob{
		svc: &svc.ServiceContext{
			Config: contentconfig.Config{Recommend: contentconfig.RecommendConfig{}},
			Redis:  client,
		},
		contentRepo: &fakeTagRefreshContentRepo{rows: []*model.ZfeedContent{
			{
				ID:          contentID,
				PublishedAt: &publishedAt,
			},
		}},
		now: func() time.Time { return now },
	}

	got, err := job.Run(context.Background(), xxljob.TriggerParam{
		ExecutorParams: `{"windowDays":7,"pageSize":10}`,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "ok refreshed=1" {
		t.Fatalf("Run result = %q, want ok refreshed=1", got)
	}

	score, err := store.ZScore(redisconsts.BuildRecommendTagIndexKey("go"), member)
	if err != nil {
		t.Fatalf("tag index score: %v", err)
	}
	if score <= 0 || score >= 999 {
		t.Fatalf("tag index score = %v, want positive decayed score below old published timestamp score", score)
	}
}

func TestParseParams(t *testing.T) {
	got := parseParams(`{"windowDays":3,"pageSize":50,"batchSize":20,"lockTtl":30}`)
	if got.WindowDays != 3 || got.PageSize != 50 || got.BatchSize != 20 || got.LockTTL != 30 {
		t.Fatalf("parseParams() = %+v", got)
	}
	if got := parseParams("{bad json"); got != (Params{}) {
		t.Fatalf("parseParams(invalid) = %+v, want zero", got)
	}
}
