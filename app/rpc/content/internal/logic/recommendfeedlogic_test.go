package logic

import "testing"

func TestParseHotFeedLuaResultSuccess(t *testing.T) {
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

func TestParseHotFeedLuaResultMiss(t *testing.T) {
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

func TestParseHotFeedLuaResultInvalid(t *testing.T) {
	if _, _, err := parseHotFeedLuaResult([]interface{}{int64(1)}); err == nil {
		t.Fatalf("expected error for invalid lua result shape")
	}
}

func TestParseHotFeedLuaResultInvalidNextCursor(t *testing.T) {
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

func TestMapHotFeedCacheError(t *testing.T) {
	cases := []struct {
		name string
		in   CacheResult
		want string
	}{
		{
			name: "cache miss",
			in:   cacheMiss,
			want: "热榜缓存不存在",
		},
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
