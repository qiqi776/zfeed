package svc

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"
)

func TestDelayedCacheInvalidatorDeletesKey(t *testing.T) {
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	invalidator := NewDelayedCacheInvalidator(redisClient, 20*time.Millisecond, 1, 16)
	defer invalidator.Close()

	const cacheKey = "count:value:10:10:9001"
	if err := redisClient.SetexCtx(context.Background(), cacheKey, "stale", 300); err != nil {
		t.Fatalf("seed cache: %v", err)
	}
	if ok := invalidator.Schedule(cacheKey, "count cache"); !ok {
		t.Fatal("schedule should succeed")
	}

	time.Sleep(60 * time.Millisecond)
	if value, err := redisClient.GetCtx(context.Background(), cacheKey); err != nil {
		t.Fatalf("read delayed delete key: %v", err)
	} else if value != "" {
		t.Fatalf("cache should be deleted, got %q", value)
	}
}

func TestDelayedCacheInvalidatorClosePreventsFutureSchedule(t *testing.T) {
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	invalidator := NewDelayedCacheInvalidator(redisClient, 20*time.Millisecond, 1, 16)
	invalidator.Close()

	if ok := invalidator.Schedule("count:value:10:10:9002", "count cache"); ok {
		t.Fatal("schedule should fail after close")
	}
}

func TestDelayedCacheInvalidatorDropsWhenQueueIsFull(t *testing.T) {
	store := miniredis.RunT(t)
	redisClient := gzredis.MustNewRedis(gzredis.RedisConf{
		Host: store.Addr(),
		Type: "node",
	})

	invalidator := &DelayedCacheInvalidator{
		redis:  redisClient,
		delay:  time.Second,
		tasks:  make(chan delayedCacheDeleteTask, 1),
		stopCh: make(chan struct{}),
	}

	invalidator.tasks <- delayedCacheDeleteTask{cacheKey: "occupied", desc: "occupied"}
	start := time.Now()
	if ok := invalidator.Schedule("count:value:10:10:9003", "count cache"); ok {
		t.Fatal("schedule should drop when queue is full")
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("schedule should not block when queue is full")
	}
	if len(invalidator.tasks) != 1 {
		t.Fatalf("queue length = %d, want 1", len(invalidator.tasks))
	}
}
