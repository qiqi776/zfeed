package feedlogic

import (
	"strconv"
	"testing"

	contentpb "zfeed/app/rpc/content/content"
)

var (
	benchmarkFollowItems []*contentpb.FollowFeedItem
	benchmarkHotFeed     *hotFeedResult
	benchmarkExists      bool
)

func BenchmarkContentItemsToFollowItems(b *testing.B) {
	items := make([]*contentpb.ContentItem, 0, 50)
	for i := 0; i < 50; i++ {
		items = append(items, &contentpb.ContentItem{
			ContentId:    int64(900000 + i),
			ContentType:  contentpb.ContentType_ARTICLE,
			AuthorId:     int64(10000 + i%10),
			AuthorName:   "bench_author",
			AuthorAvatar: "https://example.com/avatar.png",
			Title:        "bench title",
			CoverUrl:     "https://example.com/cover.png",
			PublishedAt:  1775553600,
			IsLiked:      i%2 == 0,
			LikeCount:    int64(i * 3),
		})
	}

	b.ReportAllocs()
	for b.Loop() {
		benchmarkFollowItems = ContentItemsToFollowItems(items)
	}
}

func BenchmarkParseHotFeedLuaResult(b *testing.B) {
	res := []interface{}{int64(1), int64(1), "900099", "snapshot-bench"}
	for i := 0; i < 50; i++ {
		res = append(res, strconv.FormatInt(int64(900000+i), 10))
	}

	b.ReportAllocs()
	for b.Loop() {
		var err error
		benchmarkHotFeed, benchmarkExists, err = parseHotFeedLuaResult(res)
		if err != nil {
			b.Fatal(err)
		}
	}
}
