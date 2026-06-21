package commentlogic

import (
	"context"
	"math"
	"reflect"
	"strconv"
	"testing"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/interaction/interaction"
	rediskey "zfeed/app/rpc/interaction/internal/common/consts/redis"
)

func TestBuildCmtCachedResult(t *testing.T) {
	itemMap := map[int64]*interaction.CommentItem{
		3: {CommentId: 3},
		2: {CommentId: 2},
		1: {CommentId: 1},
	}

	tests := []struct {
		name           string
		ids            []int64
		itemMap        map[int64]*interaction.CommentItem
		pageSize       uint32
		wantIDs        []int64
		wantNextCursor int64
		wantHasMore    bool
		wantOK         bool
	}{
		{
			name:     "empty cache index is a complete empty page",
			ids:      nil,
			itemMap:  itemMap,
			pageSize: 2,
			wantIDs:  []int64{},
			wantOK:   true,
		},
		{
			name:     "missing cached item invalidates result",
			ids:      []int64{3, 2},
			itemMap:  map[int64]*interaction.CommentItem{3: {CommentId: 3}},
			pageSize: 2,
		},
		{
			name:     "complete page without lookahead",
			ids:      []int64{3, 2},
			itemMap:  itemMap,
			pageSize: 2,
			wantIDs:  []int64{3, 2},
			wantOK:   true,
		},
		{
			name:           "trims lookahead and returns cursor",
			ids:            []int64{3, 2, 1},
			itemMap:        itemMap,
			pageSize:       2,
			wantIDs:        []int64{3, 2},
			wantNextCursor: 2,
			wantHasMore:    true,
			wantOK:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, nextCursor, hasMore, ok := buildCmtCachedResult(tt.ids, tt.itemMap, tt.pageSize)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got := commentItemIDs(items); !reflect.DeepEqual(got, tt.wantIDs) {
				t.Fatalf("item ids = %v, want %v", got, tt.wantIDs)
			}
			if nextCursor != tt.wantNextCursor {
				t.Fatalf("nextCursor = %d, want %d", nextCursor, tt.wantNextCursor)
			}
			if hasMore != tt.wantHasMore {
				t.Fatalf("hasMore = %v, want %v", hasMore, tt.wantHasMore)
			}
		})
	}
}

func TestCommentCacheSerialization(t *testing.T) {
	item := &interaction.CommentItem{
		CommentId: 1,
		ContentId: 2,
		UserId:    3,
		Comment:   "hello",
	}

	payload, err := marshalCommentItem(item)
	if err != nil {
		t.Fatalf("marshalCommentItem returned error: %v", err)
	}
	cloned, err := unmarshalCommentItem(payload)
	if err != nil {
		t.Fatalf("unmarshalCommentItem returned error: %v", err)
	}
	if !reflect.DeepEqual(cloned, item) {
		t.Fatalf("unmarshal item = %+v, want %+v", cloned, item)
	}

	emptyPayload, err := marshalCommentItem(nil)
	if err != nil {
		t.Fatalf("marshal nil returned error: %v", err)
	}
	if emptyPayload != "" {
		t.Fatalf("marshal nil payload = %q, want empty", emptyPayload)
	}

	emptyItem, err := unmarshalCommentItem("")
	if err != nil {
		t.Fatalf("unmarshal empty returned error: %v", err)
	}
	if emptyItem != nil {
		t.Fatalf("unmarshal empty item = %+v, want nil", emptyItem)
	}

	if _, err := unmarshalCommentItem("{"); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestStringifyLuaValue(t *testing.T) {
	tests := []struct {
		name   string
		raw    any
		want   string
		wantOK bool
	}{
		{name: "nil", raw: nil},
		{name: "string", raw: "payload", want: "payload", wantOK: true},
		{name: "bytes", raw: []byte("payload"), want: "payload", wantOK: true},
		{name: "unsupported", raw: 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := stringifyLuaValue(tt.raw)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("stringifyLuaValue() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestMaxScore(t *testing.T) {
	tests := []struct {
		name   string
		cursor int64
		want   int64
	}{
		{name: "zero uses max int", cursor: 0, want: math.MaxInt64},
		{name: "negative uses max int", cursor: -1, want: math.MaxInt64},
		{name: "min int clamps to zero", cursor: math.MinInt64, want: 0},
		{name: "positive decrements", cursor: 42, want: 41},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maxScore(tt.cursor); got != tt.want {
				t.Fatalf("maxScore(%d) = %d, want %d", tt.cursor, got, tt.want)
			}
		})
	}
}

func TestReadCmtCachedIndexIDs(t *testing.T) {
	store, redisClient := newCommentCacheRedis(t)
	ctx := context.Background()
	key := "comment:list:ARTICLE:100"

	ids, exists, hasInvalid, err := readCmtCachedIndexIDs(ctx, nil, key, 0, 2)
	if err != nil {
		t.Fatalf("nil redis returned error: %v", err)
	}
	if ids != nil || exists || hasInvalid {
		t.Fatalf("nil redis result = (%v,%v,%v), want nil,false,false", ids, exists, hasInvalid)
	}

	ids, exists, hasInvalid, err = readCmtCachedIndexIDs(ctx, redisClient, "", 0, 2)
	if err != nil {
		t.Fatalf("empty key returned error: %v", err)
	}
	if ids != nil || exists || hasInvalid {
		t.Fatalf("empty key result = (%v,%v,%v), want nil,false,false", ids, exists, hasInvalid)
	}

	ids, exists, hasInvalid, err = readCmtCachedIndexIDs(ctx, redisClient, key, 0, 2)
	if err != nil {
		t.Fatalf("missing key returned error: %v", err)
	}
	if exists || hasInvalid {
		t.Fatalf("missing key result = (%v,%v,%v), want nil,false,false", ids, exists, hasInvalid)
	}

	store.ZAdd(key, 30, "30")
	store.ZAdd(key, 20, "20")
	store.ZAdd(key, 10, "10")

	ids, exists, hasInvalid, err = readCmtCachedIndexIDs(ctx, redisClient, key, 0, 2)
	if err != nil {
		t.Fatalf("read cached ids: %v", err)
	}
	if !exists {
		t.Fatal("exists = false, want true")
	}
	if hasInvalid {
		t.Fatal("hasInvalid = true, want false")
	}
	if !reflect.DeepEqual(ids, []int64{30, 20, 10}) {
		t.Fatalf("ids = %v, want [30 20 10]", ids)
	}

	ids, exists, hasInvalid, err = readCmtCachedIndexIDs(ctx, redisClient, key, 20, 2)
	if err != nil {
		t.Fatalf("read cursor page: %v", err)
	}
	if !exists || hasInvalid || !reflect.DeepEqual(ids, []int64{10}) {
		t.Fatalf("cursor result = (%v,%v,%v), want [10],true,false", ids, exists, hasInvalid)
	}

	store.ZAdd(key, 9, "bad-id")
	ids, exists, hasInvalid, err = readCmtCachedIndexIDs(ctx, redisClient, key, 10, 2)
	if err != nil {
		t.Fatalf("read bad member page: %v", err)
	}
	if !exists || !hasInvalid || len(ids) != 0 {
		t.Fatalf("bad member result = (%v,%v,%v), want empty,true,true", ids, exists, hasInvalid)
	}
}

func TestReadCmtCachedItems(t *testing.T) {
	_, redisClient := newCommentCacheRedis(t)
	ctx := context.Background()
	item1 := &interaction.CommentItem{CommentId: 1, Comment: "one"}
	item3 := &interaction.CommentItem{CommentId: 3, Comment: "three"}
	payload1, err := marshalCommentItem(item1)
	if err != nil {
		t.Fatalf("marshal item1: %v", err)
	}
	payload3, err := marshalCommentItem(item3)
	if err != nil {
		t.Fatalf("marshal item3: %v", err)
	}

	if err := redisClient.SetCtx(ctx, rediskey.BuildCommentItemKey("1"), payload1); err != nil {
		t.Fatalf("seed item1 cache: %v", err)
	}
	if err := redisClient.SetCtx(ctx, rediskey.BuildCommentItemKey("2"), "{"); err != nil {
		t.Fatalf("seed bad item cache: %v", err)
	}
	if err := redisClient.SetCtx(ctx, rediskey.BuildCommentItemKey("3"), payload3); err != nil {
		t.Fatalf("seed item3 cache: %v", err)
	}

	itemMap, missIDs, err := readCmtCachedItems(ctx, redisClient, []int64{0, 1, 2, 3, 3, 4})
	if err != nil {
		t.Fatalf("readCmtCachedItems returned error: %v", err)
	}
	if got := sortedCommentMapKeys(itemMap); !reflect.DeepEqual(got, []int64{1, 3}) {
		t.Fatalf("cached keys = %v, want [1 3]", got)
	}
	if !reflect.DeepEqual(missIDs, []int64{2, 4}) {
		t.Fatalf("missIDs = %v, want [2 4]", missIDs)
	}

	emptyMap, emptyMiss, err := readCmtCachedItems(ctx, nil, []int64{1, 1, 0, 2})
	if err != nil {
		t.Fatalf("nil redis returned error: %v", err)
	}
	if len(emptyMap) != 0 || !reflect.DeepEqual(emptyMiss, []int64{1, 2}) {
		t.Fatalf("nil redis result = map:%v miss:%v, want empty map and [1 2]", emptyMap, emptyMiss)
	}
}

func TestCommentCacheHelpers(t *testing.T) {
	store, redisClient := newCommentCacheRedis(t)
	ctx := context.Background()
	logger := logx.WithContext(ctx)
	item := &interaction.CommentItem{CommentId: 42, Comment: "cached"}
	listKey := rediskey.BuildCommentListKey(interaction.Scene_ARTICLE.String(), "9001")
	itemKey := rediskey.BuildCommentItemKey("42")

	cmtCacheItems(ctx, logger, redisClient, []*interaction.CommentItem{nil, {CommentId: 0}, item})
	gotRaw, err := redisClient.GetCtx(ctx, itemKey)
	if err != nil {
		t.Fatalf("read cached item: %v", err)
	}
	gotItem, err := unmarshalCommentItem(gotRaw)
	if err != nil {
		t.Fatalf("unmarshal cached item: %v", err)
	}
	if gotItem.GetCommentId() != item.GetCommentId() || gotItem.GetComment() != item.GetComment() {
		t.Fatalf("cached item = %+v, want %+v", gotItem, item)
	}

	cmtCacheItemsAndIndex(ctx, logger, redisClient, listKey, []*interaction.CommentItem{item})
	members, err := store.ZMembers(listKey)
	if err != nil {
		t.Fatalf("read list zset: %v", err)
	}
	if !reflect.DeepEqual(members, []string{"42"}) {
		t.Fatalf("list members = %v, want [42]", members)
	}
	if ttl := store.TTL(listKey); ttl <= 0 {
		t.Fatalf("list ttl = %v, want positive", ttl)
	}

	invalidateCmtCacheKey(ctx, logger, redisClient, "", itemKey, itemKey, listKey)
	if store.Exists(itemKey) || store.Exists(listKey) {
		t.Fatalf("cache keys still exist after invalidation: item=%v list=%v", store.Exists(itemKey), store.Exists(listKey))
	}

	lockKey := rediskey.BuildCommentListLockKey(interaction.Scene_ARTICLE.String(), strconv.FormatInt(9001, 10))
	locked, err := tryAcquireCommentRebuild(ctx, nil, lockKey)
	if err != nil {
		t.Fatalf("nil redis lock returned error: %v", err)
	}
	if locked {
		t.Fatal("nil redis lock = true, want false")
	}
	locked, err = tryAcquireCommentRebuild(ctx, redisClient, lockKey)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if !locked {
		t.Fatal("first lock acquire = false, want true")
	}
	locked, err = tryAcquireCommentRebuild(ctx, redisClient, lockKey)
	if err != nil {
		t.Fatalf("second acquire lock: %v", err)
	}
	if locked {
		t.Fatal("second lock acquire = true, want false")
	}
	releaseCommentRebuild(ctx, logger, redisClient, lockKey)
	if store.Exists(lockKey) {
		t.Fatal("lock key still exists after release")
	}
}

func TestCommentCacheHelperErrors(t *testing.T) {
	ctx := context.Background()
	logger := logx.WithContext(ctx)
	_, redisClient := newCommentCacheRedis(t)
	errorRedis := newErrorCommentRedis(t)

	cmtCacheItems(ctx, logger, nil, []*interaction.CommentItem{{CommentId: 8101}})
	cmtCacheItems(ctx, logger, redisClient, []*interaction.CommentItem{nil, {CommentId: 0}})
	cmtCacheItems(ctx, logger, errorRedis, []*interaction.CommentItem{{CommentId: 8102, Comment: "write error"}})

	cmtCacheItemsAndIndex(ctx, logger, nil, "comment:list:error", []*interaction.CommentItem{{CommentId: 8103}})
	cmtCacheItemsAndIndex(ctx, logger, redisClient, "", []*interaction.CommentItem{{CommentId: 8104}})
	cmtCacheItemsAndIndex(ctx, logger, redisClient, "comment:list:empty", []*interaction.CommentItem{nil, {CommentId: 0}})
	cmtCacheItemsAndIndex(ctx, logger, errorRedis, "comment:list:error", []*interaction.CommentItem{{CommentId: 8105, Comment: "index error"}})

	invalidateCmtCacheKey(ctx, logger, nil, "comment:item:8101")
	invalidateCmtCacheKey(ctx, logger, redisClient, "", "")
	invalidateCmtCacheKey(ctx, logger, errorRedis, "comment:item:8102")

	releaseCommentRebuild(ctx, logger, nil, "comment:lock:error")
	releaseCommentRebuild(ctx, logger, redisClient, "")
	releaseCommentRebuild(ctx, logger, errorRedis, "comment:lock:error")
}
