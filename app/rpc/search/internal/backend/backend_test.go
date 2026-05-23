package backend

import (
	"context"
	"testing"
	"time"
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
