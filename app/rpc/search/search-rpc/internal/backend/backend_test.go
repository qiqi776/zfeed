package backend

import (
	"context"
	"testing"
	"time"

	"zfeed/app/rpc/search/search-rpc/internal/repositories"
)

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults to mysql", in: "", want: NameMySQL},
		{name: "mysql", in: "mysql", want: NameMySQL},
		{name: "engine", in: "engine", want: NameEngine},
		{name: "unknown falls back to mysql", in: "unknown", want: NameMySQL},
		{name: "case insensitive", in: " MySQL ", want: NameMySQL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeName(tt.in); got != tt.want {
				t.Fatalf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFactoryUsesMySQLWhenEngineTrafficIsZero(t *testing.T) {
	factory := NewFactory(nil, NameEngine, 0)
	if got := factory.ConfiguredBackend(); got != NameEngine {
		t.Fatalf("ConfiguredBackend = %q, want %q", got, NameEngine)
	}
	if got := factory.EffectiveBackend(); got != NameMySQL {
		t.Fatalf("EffectiveBackend = %q, want %q", got, NameMySQL)
	}
	if got := factory.Backend(nil).Name(); got != NameMySQL {
		t.Fatalf("Backend name = %q, want %q", got, NameMySQL)
	}
}

func TestFactoryUsesEngineWhenTrafficIsFull(t *testing.T) {
	factory := NewFactory(nil, NameEngine, 100)
	if got := factory.EffectiveBackend(); got != NameEngine {
		t.Fatalf("EffectiveBackend = %q, want %q", got, NameEngine)
	}
	if got := factory.Backend(nil).Name(); got != NameEngine {
		t.Fatalf("Backend name = %q, want %q", got, NameEngine)
	}
}

func TestCompareBackendShadowDoesNotBlockPrimaryResult(t *testing.T) {
	primary := blockingSearchBackend{name: NameMySQL}
	shadow := blockingSearchBackend{name: NameEngine, delay: 200 * time.Millisecond}
	compare := NewCompareBackend(primary, shadow)
	compare.shadowTimeout = 10 * time.Millisecond

	start := time.Now()
	if _, err := compare.SearchUsers(context.Background(), "alice", 0, 10); err != nil {
		t.Fatalf("SearchUsers returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("SearchUsers blocked on shadow backend for %s", elapsed)
	}
}

func TestOverlapRatio(t *testing.T) {
	tests := []struct {
		name  string
		left  []int64
		right []int64
		want  float64
	}{
		{name: "both empty", want: 1},
		{name: "left empty", right: []int64{1}, want: 0},
		{name: "right empty", left: []int64{1}, want: 0},
		{name: "partial overlap", left: []int64{1, 2, 3}, right: []int64{2, 3, 4}, want: 2.0 / 3.0},
		{name: "right longer", left: []int64{1, 2}, right: []int64{1, 2, 3, 4}, want: 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := overlapRatio(tt.left, tt.right); got != tt.want {
				t.Fatalf("overlapRatio(%v, %v) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestTopContentIDsRespectsLimitAndSkipsInvalidIDs(t *testing.T) {
	rows := []repositories.SearchContentRow{
		{ContentID: 1001},
		{ContentID: 0},
		{ContentID: 1003},
	}
	got := topContentIDs(rows, 3)
	want := []int64{1001, 1003}
	if len(got) != len(want) {
		t.Fatalf("topContentIDs len = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("topContentIDs = %v, want %v", got, want)
		}
	}
}

type blockingSearchBackend struct {
	name  string
	delay time.Duration
}

func (b blockingSearchBackend) Name() string {
	return b.name
}

func (b blockingSearchBackend) SearchUsers(context.Context, string, int64, int) (SearchUsersResult, error) {
	if b.delay > 0 {
		time.Sleep(b.delay)
	}
	return SearchUsersResult{}, nil
}

func (b blockingSearchBackend) SearchContents(context.Context, string, string, int64, int) (SearchContentsResult, error) {
	if b.delay > 0 {
		time.Sleep(b.delay)
	}
	return SearchContentsResult{}, nil
}
