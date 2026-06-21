package commentcache

import (
	"encoding/json"
	"testing"

	"zfeed/app/rpc/interaction/interaction"
)

var (
	benchmarkCommentJSON    []byte
	benchmarkCommentItem    *interaction.CommentItem
	benchmarkCommentItems   []*interaction.CommentItem
	benchmarkCommentItemMap map[int64]*interaction.CommentItem
)

func BenchmarkCommentCacheJSONMarshal(b *testing.B) {
	item := newCommentItem(700001)

	b.ReportAllocs()
	for b.Loop() {
		var err error
		benchmarkCommentJSON, err = json.Marshal(item)
		if err != nil {
			b.Fatalf("marshal comment item: %v", err)
		}
	}
}

func BenchmarkCommentCacheJSONUnmarshal(b *testing.B) {
	payload, err := json.Marshal(newCommentItem(700001))
	if err != nil {
		b.Fatalf("prepare comment payload: %v", err)
	}

	b.ReportAllocs()
	for b.Loop() {
		var item interaction.CommentItem
		if err := json.Unmarshal(payload, &item); err != nil {
			b.Fatalf("unmarshal comment item: %v", err)
		}
		benchmarkCommentItem = &item
	}
}

func BenchmarkCommentCacheBatchAssemble(b *testing.B) {
	ids := make([]int64, 100)
	itemMap := make(map[int64]*interaction.CommentItem, len(ids))
	for i := range ids {
		id := int64(700000 + i)
		ids[i] = id
		itemMap[id] = newCommentItem(id)
	}

	b.ReportAllocs()
	for b.Loop() {
		benchmarkCommentItems = assembleCommentItems(ids, itemMap, 50)
		if len(benchmarkCommentItems) != 50 {
			b.Fatalf("assembled items = %d, want 50", len(benchmarkCommentItems))
		}
	}
}

func BenchmarkCommentCacheBuildItemMap(b *testing.B) {
	items := make([]*interaction.CommentItem, 100)
	for i := range items {
		items[i] = newCommentItem(int64(700000 + i))
	}

	b.ReportAllocs()
	for b.Loop() {
		benchmarkCommentItemMap = buildCommentItemMap(items)
		if len(benchmarkCommentItemMap) != len(items) {
			b.Fatalf("item map = %d, want %d", len(benchmarkCommentItemMap), len(items))
		}
	}
}

func newCommentItem(id int64) *interaction.CommentItem {
	return &interaction.CommentItem{
		CommentId:     id,
		ContentId:     900001,
		UserId:        10001 + id%17,
		ReplyToUserId: 10002 + id%11,
		ParentId:      id - 1,
		RootId:        id - id%10,
		Comment:       "bench comment payload with enough text to resemble a normal user comment",
		CreatedAt:     1775553600 + id%3600,
		Status:        10,
		UserName:      "bench_user",
		UserAvatar:    "https://example.com/bench/avatar.png",
		ReplyCount:    id % 13,
	}
}

func assembleCommentItems(
	ids []int64,
	itemMap map[int64]*interaction.CommentItem,
	limit int,
) []*interaction.CommentItem {
	if limit <= 0 || len(ids) == 0 {
		return []*interaction.CommentItem{}
	}
	if limit > len(ids) {
		limit = len(ids)
	}

	items := make([]*interaction.CommentItem, 0, limit)
	for _, id := range ids {
		item := itemMap[id]
		if item == nil {
			continue
		}
		items = append(items, item)
		if len(items) == limit {
			break
		}
	}
	return items
}

func buildCommentItemMap(items []*interaction.CommentItem) map[int64]*interaction.CommentItem {
	itemMap := make(map[int64]*interaction.CommentItem, len(items))
	for _, item := range items {
		if item == nil || item.GetCommentId() <= 0 {
			continue
		}
		itemMap[item.GetCommentId()] = item
	}
	return itemMap
}
