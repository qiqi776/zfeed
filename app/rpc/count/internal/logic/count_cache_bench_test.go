package logic

import (
	"testing"

	"zfeed/app/rpc/count/count"
)

var (
	benchmarkCountCacheKey string
	benchmarkProfileJSON   string
	benchmarkProfileCounts *count.GetUserProfileCountsRes
)

func BenchmarkBuildCountValueCacheKey(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkCountCacheKey = buildCountValueCacheKey(count.BizType_LIKE, count.TargetType_CONTENT, 900001)
	}
}

func BenchmarkMarshalUserProfileCounts(b *testing.B) {
	value := &count.GetUserProfileCountsRes{
		FollowingCount: 120,
		FollowedCount:  340,
		LikeCount:      560,
		FavoriteCount:  780,
	}

	b.ReportAllocs()
	for b.Loop() {
		var err error
		benchmarkProfileJSON, err = marshalUserProfileCounts(value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalUserProfileCounts(b *testing.B) {
	payload := `{"following_count":120,"followed_count":340,"like_count":560,"favorite_count":780}`

	b.ReportAllocs()
	for b.Loop() {
		var err error
		benchmarkProfileCounts, err = unmarshalUserProfileCounts(payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}
