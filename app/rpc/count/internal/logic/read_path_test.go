package logic

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/model"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
)

func TestGetCountCacheHitUsesCache(t *testing.T) {
	svcCtx, _, db := newCountLogicTestServiceContext(t)
	ctx := context.Background()

	seedCountValue(t, db, count.BizType_LIKE, count.TargetType_CONTENT, 1001, 2001, 3)
	cacheKey := buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 1001)
	if err := svcCtx.Redis.SetexCtx(ctx, cacheKey, "9", 300); err != nil {
		t.Fatalf("seed count cache: %v", err)
	}

	logic := NewGetCountLogic(ctx, svcCtx)
	resp, err := logic.GetCount(&count.GetCountReq{
		BizType:    count.BizType_LIKE,
		TargetType: count.TargetType_CONTENT,
		TargetId:   1001,
	})
	if err != nil {
		t.Fatalf("get count: %v", err)
	}
	if resp.GetValue() != 9 {
		t.Fatalf("count = %d, want 9", resp.GetValue())
	}
}

func TestGetCountCacheMissRebuildsCache(t *testing.T) {
	svcCtx, _, db := newCountLogicTestServiceContext(t)
	ctx := context.Background()

	seedCountValue(t, db, count.BizType_LIKE, count.TargetType_CONTENT, 1002, 2002, 7)

	logic := NewGetCountLogic(ctx, svcCtx)
	resp, err := logic.GetCount(&count.GetCountReq{
		BizType:    count.BizType_LIKE,
		TargetType: count.TargetType_CONTENT,
		TargetId:   1002,
	})
	if err != nil {
		t.Fatalf("get count: %v", err)
	}
	if resp.GetValue() != 7 {
		t.Fatalf("count = %d, want 7", resp.GetValue())
	}

	cacheKey := buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 1002)
	if cached, err := svcCtx.Redis.GetCtx(ctx, cacheKey); err != nil {
		t.Fatalf("read rebuilt count cache: %v", err)
	} else if cached != "7" {
		t.Fatalf("rebuilt count cache = %q, want %q", cached, "7")
	}
}

func TestBatchGetCountLoadsCacheAndDB(t *testing.T) {
	svcCtx, _, db := newCountLogicTestServiceContext(t)
	ctx := context.Background()

	seedCountValue(t, db, count.BizType_LIKE, count.TargetType_CONTENT, 1102, 2102, 3)
	seedCountValue(t, db, count.BizType_FAVORITE, count.TargetType_CONTENT, 1103, 2103, 4)

	cachedKey := buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 1101)
	if err := svcCtx.Redis.SetexCtx(ctx, cachedKey, "11", 300); err != nil {
		t.Fatalf("seed batch cache: %v", err)
	}

	logic := NewBatchGetCountLogic(ctx, svcCtx)
	resp, err := logic.BatchGetCount(&count.BatchGetCountReq{
		Keys: []*count.CountKey{
			{BizType: count.BizType_LIKE, TargetType: count.TargetType_CONTENT, TargetId: 1101},
			{BizType: count.BizType_LIKE, TargetType: count.TargetType_CONTENT, TargetId: 1102},
			{BizType: count.BizType_FAVORITE, TargetType: count.TargetType_CONTENT, TargetId: 1103},
			{BizType: count.BizType_FOLLOWED, TargetType: count.TargetType_USER, TargetId: 1104},
		},
	})
	if err != nil {
		t.Fatalf("batch get count: %v", err)
	}
	if len(resp.GetItems()) != 4 {
		t.Fatalf("items len = %d, want 4", len(resp.GetItems()))
	}

	got := []int64{
		resp.GetItems()[0].GetValue(),
		resp.GetItems()[1].GetValue(),
		resp.GetItems()[2].GetValue(),
		resp.GetItems()[3].GetValue(),
	}
	want := []int64{11, 3, 4, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item[%d] = %d, want %d", i, got[i], want[i])
		}
	}

	if cached, err := svcCtx.Redis.GetCtx(ctx, buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 1102)); err != nil {
		t.Fatalf("read rebuilt batch cache: %v", err)
	} else if cached != "3" {
		t.Fatalf("rebuilt batch cache = %q, want %q", cached, "3")
	}
	if cached, err := svcCtx.Redis.GetCtx(ctx, buildCountValueCacheKey(count.BizType_FOLLOWED, count.TargetType_USER, 1104)); err != nil {
		t.Fatalf("read zero batch cache: %v", err)
	} else if cached != "0" {
		t.Fatalf("zero batch cache = %q, want %q", cached, "0")
	}
}

func TestGetUserProfileCountsRebuildsCacheFromDB(t *testing.T) {
	svcCtx, _, db := newCountLogicTestServiceContext(t)
	ctx := context.Background()

	seedCountValue(t, db, count.BizType_LIKE, count.TargetType_CONTENT, 1201, 3001, 2)
	seedCountValue(t, db, count.BizType_LIKE, count.TargetType_CONTENT, 1202, 3001, 3)
	seedCountValue(t, db, count.BizType_FAVORITE, count.TargetType_CONTENT, 1203, 3001, 4)
	seedCountValue(t, db, count.BizType_FOLLOWING, count.TargetType_USER, 3001, 0, 5)
	seedCountValue(t, db, count.BizType_FOLLOWED, count.TargetType_USER, 3001, 0, 6)

	logic := NewGetUserProfileCountsLogic(ctx, svcCtx)
	resp, err := logic.GetUserProfileCounts(&count.GetUserProfileCountsReq{UserId: 3001})
	if err != nil {
		t.Fatalf("get user profile counts: %v", err)
	}
	if resp.GetLikeCount() != 5 || resp.GetFavoriteCount() != 4 || resp.GetFollowingCount() != 5 || resp.GetFollowedCount() != 6 {
		t.Fatalf("unexpected user profile counts: %+v", resp)
	}

	cacheKey := buildUserProfileCountsCacheKey(3001)
	if cached, err := svcCtx.Redis.GetCtx(ctx, cacheKey); err != nil {
		t.Fatalf("read rebuilt user profile cache: %v", err)
	} else if cached == "" {
		t.Fatal("user profile counts cache should be rebuilt")
	}
}

func TestGetCountConcurrentRebuildHitsDBOnce(t *testing.T) {
	svcCtx, _, _ := newCountLogicTestServiceContext(t)
	ctx := context.Background()

	repo := &stubCountValueRepository{
		getValue:  8,
		getDelay:  50 * time.Millisecond,
		batchRows: map[int64]*model.ZfeedCountValue{},
	}

	logic := NewGetCountLogic(ctx, svcCtx)
	logic.countRepo = repo

	const goroutines = 8
	errCh := make(chan error, goroutines)
	valueCh := make(chan int64, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := logic.GetCount(&count.GetCountReq{
				BizType:    count.BizType_LIKE,
				TargetType: count.TargetType_CONTENT,
				TargetId:   1301,
			})
			if err != nil {
				errCh <- err
				return
			}
			valueCh <- resp.GetValue()
		}()
	}
	wg.Wait()
	close(errCh)
	close(valueCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent get count failed: %v", err)
		}
	}
	for value := range valueCh {
		if value != 8 {
			t.Fatalf("concurrent get count value = %d, want 8", value)
		}
	}
	if repo.getCalls != 1 {
		t.Fatalf("db get calls = %d, want 1", repo.getCalls)
	}

	cacheKey := buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 1301)
	if cached, err := svcCtx.Redis.GetCtx(ctx, cacheKey); err != nil {
		t.Fatalf("read concurrent rebuilt cache: %v", err)
	} else if cached != "8" {
		t.Fatalf("concurrent rebuilt cache = %q, want %q", cached, "8")
	}
}

func TestCountOperatorDelayedSecondDeleteRemovesRewrittenCache(t *testing.T) {
	svcCtx, _, _ := newCountLogicTestServiceContext(t)
	ctx := context.Background()

	operator := NewCountOperator(ctx, svcCtx)

	countKey := buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 1401)
	if err := svcCtx.Redis.SetexCtx(ctx, countKey, "stale", 300); err != nil {
		t.Fatalf("seed count key: %v", err)
	}
	operator.InvalidateCountCache(count.BizType_LIKE, count.TargetType_CONTENT, 1401)
	if err := svcCtx.Redis.SetexCtx(ctx, countKey, "rebuilt-old", 300); err != nil {
		t.Fatalf("rewrite count key before delayed delete: %v", err)
	}
	time.Sleep(delayedCacheInvalidateDelay + 100*time.Millisecond)
	if cached, err := svcCtx.Redis.GetCtx(ctx, countKey); err != nil {
		t.Fatalf("read delayed deleted count key: %v", err)
	} else if cached != "" {
		t.Fatalf("count key should be deleted by delayed invalidation, got %q", cached)
	}

	profileKey := buildUserProfileCountsCacheKey(1402)
	if err := svcCtx.Redis.SetexCtx(ctx, profileKey, `{"like_count":1}`, 300); err != nil {
		t.Fatalf("seed profile key: %v", err)
	}
	operator.InvalidateUserProfileCountsCache(1402)
	if err := svcCtx.Redis.SetexCtx(ctx, profileKey, `{"like_count":2}`, 300); err != nil {
		t.Fatalf("rewrite profile key before delayed delete: %v", err)
	}
	time.Sleep(delayedCacheInvalidateDelay + 100*time.Millisecond)
	if cached, err := svcCtx.Redis.GetCtx(ctx, profileKey); err != nil {
		t.Fatalf("read delayed deleted profile key: %v", err)
	} else if cached != "" {
		t.Fatalf("profile key should be deleted by delayed invalidation, got %q", cached)
	}
}

func newCountLogicTestServiceContext(t *testing.T) (*svc.ServiceContext, *miniredis.Miniredis, *gorm.DB) {
	t.Helper()

	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	db, err := gorm.Open(sqlite.Open("file:count_logic_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.ZfeedCountValue{}, &model.ZfeedMqConsumeDedup{}); err != nil {
		t.Fatalf("auto migrate count models: %v", err)
	}

	svcCtx := &svc.ServiceContext{
		Redis:                   redisClient,
		MysqlDb:                 db,
		DelayedCacheInvalidator: svc.NewDelayedCacheInvalidator(redisClient, delayedCacheInvalidateDelay, 1, 64),
	}
	t.Cleanup(func() {
		svcCtx.Close()
	})
	return svcCtx, store, db
}

func seedCountValue(
	t *testing.T,
	db *gorm.DB,
	bizType count.BizType,
	targetType count.TargetType,
	targetID int64,
	ownerID int64,
	value int64,
) {
	t.Helper()

	if err := db.Create(&model.ZfeedCountValue{
		BizType:    int32(bizType),
		TargetType: int32(targetType),
		TargetID:   targetID,
		OwnerID:    ownerID,
		Value:      value,
		Version:    1,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed count value: %v", err)
	}
}

type stubCountValueRepository struct {
	mu        sync.Mutex
	getCalls  int
	getValue  int64
	getDelay  time.Duration
	batchRows map[int64]*model.ZfeedCountValue
}

func (s *stubCountValueRepository) WithTx(tx *gorm.DB) repositories.CountValueRepository {
	return s
}

func (s *stubCountValueRepository) Get(bizType int32, targetType int32, targetID int64) (*model.ZfeedCountValue, error) {
	s.mu.Lock()
	s.getCalls++
	delay := s.getDelay
	value := s.getValue
	s.mu.Unlock()

	if delay > 0 {
		time.Sleep(delay)
	}
	return &model.ZfeedCountValue{
		BizType:    bizType,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		Version:    1,
	}, nil
}

func (s *stubCountValueRepository) BatchGet(bizType int32, targetType int32, targetIDs []int64) (map[int64]*model.ZfeedCountValue, error) {
	rows := make(map[int64]*model.ZfeedCountValue, len(targetIDs))
	for _, targetID := range targetIDs {
		if row, ok := s.batchRows[targetID]; ok {
			rows[targetID] = row
		}
	}
	return rows, nil
}

func (s *stubCountValueRepository) SumByOwner(bizType int32, targetType int32, ownerID int64) (int64, error) {
	return 0, errors.New("not implemented")
}

func (s *stubCountValueRepository) ApplyDelta(
	bizType int32,
	targetType int32,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) (int64, error) {
	return 0, errors.New("not implemented")
}

func TestBuildRebuildLockKeys(t *testing.T) {
	countKey := buildCountValueRebuildLockKey(count.BizType_LIKE, count.TargetType_CONTENT, 1501)
	if countKey != redisconsts.RedisCountRebuildLockPrefix+":10:10:1501" {
		t.Fatalf("count rebuild key = %q", countKey)
	}

	profileKey := buildUserProfileCountsRebuildLockKey(1502)
	if profileKey != redisconsts.RedisUserProfileCountsRebuildLockPref+":1502" {
		t.Fatalf("user profile rebuild key = %q", profileKey)
	}
}
