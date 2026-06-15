package producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/repositories"
)

const (
	defaultUserActionOutboxRelayInterval = time.Second
	defaultUserActionOutboxRelayBatch    = 100
)

type UserActionProducer struct {
	pusher     messagePusher
	db         *gorm.DB
	maxRetries int
}

type UserActionEventProducer interface {
	SendUserAction(ctx context.Context, action event.UserActionEvent) error
}

type UserActionOutboxRelay struct {
	producer  *UserActionProducer
	interval  time.Duration
	batchSize int
	stopCh    chan struct{}
	doneCh    chan struct{}
	stopOnce  sync.Once
	mu        sync.Mutex
	started   bool
}

func NewUserActionProducer(pusher *kq.Pusher, db *gorm.DB, maxRetries int) *UserActionProducer {
	if maxRetries <= 0 {
		maxRetries = 1
	}

	return &UserActionProducer{
		pusher:     pusher,
		db:         db,
		maxRetries: maxRetries,
	}
}

func NewUserActionOutboxRelay(producer *UserActionProducer) *UserActionOutboxRelay {
	return &UserActionOutboxRelay{
		producer:  producer,
		interval:  defaultUserActionOutboxRelayInterval,
		batchSize: defaultUserActionOutboxRelayBatch,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

func (p *UserActionProducer) SendUserAction(ctx context.Context, action event.UserActionEvent) error {
	if action.EventID == "" {
		action = event.NewUserActionEvent(
			action.Action,
			action.UserID,
			action.TargetID,
			action.ContentUserID,
			action.Scene,
		)
	}
	if action.Source == "" {
		action.Source = event.UserActionSourceInteraction
	}
	if action.TargetType == "" {
		action.TargetType = event.UserActionTargetContent
	}
	if action.ContentID <= 0 {
		action.ContentID = action.TargetID
	}
	if action.OccurredAt <= 0 {
		action.OccurredAt = time.Now().Unix()
	}

	return p.enqueueEvent(ctx, action)
}

func (p *UserActionProducer) enqueueEvent(ctx context.Context, action event.UserActionEvent) error {
	if p == nil || p.db == nil || p.pusher == nil {
		return fmt.Errorf("user action producer dependencies are not ready")
	}

	body, err := action.Marshal()
	if err != nil {
		return err
	}

	repo := repositories.NewUserActionOutboxRepository(ctx, p.db)
	now := time.Now()
	if err := repo.InsertPending(action.EventID, action.Action, string(body), now); err != nil {
		return err
	}

	if err := p.dispatchPayload(ctx, string(body)); err != nil {
		logx.WithContext(ctx).Errorf("dispatch user action event failed, event_id=%s, err=%v", action.EventID, err)
		if markErr := repo.MarkRetry(action.EventID, nextUserActionRetryTime(now, 1), err.Error()); markErr != nil {
			logx.WithContext(ctx).Errorf("mark user action retry failed, event_id=%s, err=%v", action.EventID, markErr)
		}
		recordUserActionOutboxMetric(action.Action, userActionOutboxResultRetry)
		return nil
	}

	if err := repo.MarkSent(action.EventID, time.Now()); err != nil {
		logx.WithContext(ctx).Errorf("mark user action sent failed, event_id=%s, err=%v", action.EventID, err)
		recordUserActionOutboxMetric(action.Action, userActionOutboxResultMarkFailed)
		return nil
	}
	recordUserActionOutboxMetric(action.Action, userActionOutboxResultSent)
	return nil
}

func (p *UserActionProducer) dispatchDueEvents(ctx context.Context, batchSize int) {
	if p == nil || p.db == nil || p.pusher == nil || batchSize <= 0 {
		return
	}

	repo := repositories.NewUserActionOutboxRepository(ctx, p.db)
	records, err := repo.ListDuePending(batchSize, time.Now())
	if err != nil {
		logx.WithContext(ctx).Errorf("list due user action outbox events failed, err=%v", err)
		return
	}

	for _, record := range records {
		if err := p.dispatchPayload(ctx, record.Payload); err != nil {
			logx.WithContext(ctx).Errorf("replay user action event failed, event_id=%s, err=%v", record.EventID, err)
			nextRetryAt := nextUserActionRetryTime(time.Now(), record.RetryCount+1)
			if markErr := repo.MarkRetry(record.EventID, nextRetryAt, err.Error()); markErr != nil {
				logx.WithContext(ctx).Errorf("mark user action replay retry failed, event_id=%s, err=%v", record.EventID, markErr)
			}
			recordUserActionOutboxMetric(record.EventType, userActionOutboxResultRetry)
			continue
		}

		if err := repo.MarkSent(record.EventID, time.Now()); err != nil {
			logx.WithContext(ctx).Errorf("mark user action replay sent failed, event_id=%s, err=%v", record.EventID, err)
			recordUserActionOutboxMetric(record.EventType, userActionOutboxResultMarkFailed)
			continue
		}
		recordUserActionOutboxMetric(record.EventType, userActionOutboxResultReplayed)
	}
}

func (p *UserActionProducer) dispatchPayload(ctx context.Context, payload string) error {
	var lastErr error
	for i := 0; i < p.maxRetries; i++ {
		if err := p.sendRaw(ctx, payload); err == nil {
			return nil
		} else {
			lastErr = err
			logx.WithContext(ctx).Errorf("send user action event failed, retry %d/%d, err=%v", i+1, p.maxRetries, err)
			time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("send user action event failed")
}

func (p *UserActionProducer) sendRaw(ctx context.Context, payload string) error {
	return p.pusher.Push(ctx, payload)
}

func nextUserActionRetryTime(now time.Time, retryCount int) time.Time {
	if retryCount < 1 {
		retryCount = 1
	}

	backoff := time.Second * time.Duration(1<<minUserActionRetryShift(retryCount-1))
	maxBackoff := 5 * time.Minute
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return now.Add(backoff)
}

func minUserActionRetryShift(v int) int {
	if v < 0 {
		return 0
	}
	if v > 8 {
		return 8
	}
	return v
}

func (r *UserActionOutboxRelay) Start() {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	r.started = true
	r.mu.Unlock()

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		defer close(r.doneCh)

		for {
			select {
			case <-ticker.C:
				r.producer.dispatchDueEvents(context.Background(), r.batchSize)
			case <-r.stopCh:
				return
			}
		}
	}()
}

func (r *UserActionOutboxRelay) Stop() {
	r.mu.Lock()
	started := r.started
	r.mu.Unlock()
	if !started {
		return
	}

	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	<-r.doneCh
}
