package svc

import (
	"context"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type delayedCacheDeleteTask struct {
	cacheKey string
	desc     string
}

type DelayedCacheInvalidator struct {
	redis  *redis.Redis
	delay  time.Duration
	tasks  chan delayedCacheDeleteTask
	stopCh chan struct{}

	closeOnce sync.Once
	wg        sync.WaitGroup
}

func NewDelayedCacheInvalidator(redisClient *redis.Redis, delay time.Duration, workers int, queueSize int) *DelayedCacheInvalidator {
	if redisClient == nil || workers <= 0 || queueSize <= 0 || delay <= 0 {
		return nil
	}

	invalidator := &DelayedCacheInvalidator{
		redis:  redisClient,
		delay:  delay,
		tasks:  make(chan delayedCacheDeleteTask, queueSize),
		stopCh: make(chan struct{}),
	}

	for i := 0; i < workers; i++ {
		invalidator.wg.Add(1)
		go invalidator.runWorker()
	}

	return invalidator
}

func (i *DelayedCacheInvalidator) Schedule(cacheKey string, desc string) bool {
	if i == nil || cacheKey == "" {
		return false
	}

	select {
	case <-i.stopCh:
		return false
	case i.tasks <- delayedCacheDeleteTask{cacheKey: cacheKey, desc: desc}:
		return true
	default:
		// Delayed delete is best-effort and must not block the write path.
		logx.Errorf("drop delayed delete task because queue is full, key=%s, desc=%s", cacheKey, desc)
		return false
	}
}

func (i *DelayedCacheInvalidator) Close() {
	if i == nil {
		return
	}

	i.closeOnce.Do(func() {
		close(i.stopCh)
		i.wg.Wait()
	})
}

func (i *DelayedCacheInvalidator) runWorker() {
	defer i.wg.Done()

	for {
		select {
		case <-i.stopCh:
			return
		case task := <-i.tasks:
			timer := time.NewTimer(i.delay)
			select {
			case <-i.stopCh:
				if !timer.Stop() {
					<-timer.C
				}
				return
			case <-timer.C:
			}

			if _, err := i.redis.DelCtx(context.Background(), task.cacheKey); err != nil {
				logx.Errorf("delayed delete %s failed, key=%s, err=%v", task.desc, task.cacheKey, err)
			}
		}
	}
}
