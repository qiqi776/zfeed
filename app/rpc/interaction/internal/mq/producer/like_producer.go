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
	defaultLikeOutboxRelayInterval = time.Second
	defaultLikeOutboxRelayBatch    = 100
)

type messagePusher interface {
	Push(ctx context.Context, value string) error
}

type LikeProducer struct {
	pusher     messagePusher
	db         *gorm.DB
	maxRetries int
}

type EventProducer interface {
	SendLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) error
	SendCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) error
}

type LikeOutboxRelay struct {
	producer  *LikeProducer
	interval  time.Duration
	batchSize int
	stopCh    chan struct{}
	doneCh    chan struct{}
	stopOnce  sync.Once
	mu        sync.Mutex
	started   bool
}

func NewLikeProducer(pusher *kq.Pusher, db *gorm.DB, maxRetries int) *LikeProducer {
	if maxRetries <= 0 {
		maxRetries = 1
	}

	return &LikeProducer{
		pusher:     pusher,
		db:         db,
		maxRetries: maxRetries,
	}
}

func NewLikeOutboxRelay(producer *LikeProducer) *LikeOutboxRelay {
	return &LikeOutboxRelay{
		producer:  producer,
		interval:  defaultLikeOutboxRelayInterval,
		batchSize: defaultLikeOutboxRelayBatch,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

func (p *LikeProducer) SendLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) error {
	now := time.Now().UnixNano()
	likeEvent := &event.LikeEvent{
		EventID:       fmt.Sprintf("like_%d_%d_%d", userID, contentID, now),
		EventType:     event.EventTypeLike,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         scene,
		Timestamp:     now,
	}

	return p.enqueueEvent(ctx, likeEvent)
}

func (p *LikeProducer) SendCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) error {
	now := time.Now().UnixNano()
	likeEvent := &event.LikeEvent{
		EventID:       fmt.Sprintf("cancel_like_%d_%d_%d", userID, contentID, now),
		EventType:     event.EventTypeCancel,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         scene,
		Timestamp:     now,
	}

	return p.enqueueEvent(ctx, likeEvent)
}

func (p *LikeProducer) enqueueEvent(ctx context.Context, evt *event.LikeEvent) error {
	if p == nil || p.db == nil || p.pusher == nil {
		return fmt.Errorf("like producer dependencies are not ready")
	}

	body, err := evt.Marshal()
	if err != nil {
		return err
	}

	repo := repositories.NewLikeEventOutboxRepository(ctx, p.db)
	now := time.Now()
	if err := repo.InsertPending(evt.EventID, string(evt.EventType), string(body), now); err != nil {
		return err
	}

	if err := p.dispatchPayload(ctx, string(body)); err != nil {
		logx.WithContext(ctx).Errorf("dispatch like event failed, event_id=%s, err=%v", evt.EventID, err)
		if markErr := repo.MarkRetry(evt.EventID, nextLikeEventRetryTime(now, 1), err.Error()); markErr != nil {
			logx.WithContext(ctx).Errorf("mark like event retry failed, event_id=%s, err=%v", evt.EventID, markErr)
		}
		return nil
	}

	if err := repo.MarkSent(evt.EventID, time.Now()); err != nil {
		logx.WithContext(ctx).Errorf("mark like event sent failed, event_id=%s, err=%v", evt.EventID, err)
	}
	return nil
}

func (p *LikeProducer) dispatchDueEvents(ctx context.Context, batchSize int) {
	if p == nil || p.db == nil || p.pusher == nil || batchSize <= 0 {
		return
	}

	repo := repositories.NewLikeEventOutboxRepository(ctx, p.db)
	records, err := repo.ListDuePending(batchSize, time.Now())
	if err != nil {
		logx.WithContext(ctx).Errorf("list due like outbox events failed, err=%v", err)
		return
	}

	for _, record := range records {
		if err := p.dispatchPayload(ctx, record.Payload); err != nil {
			logx.WithContext(ctx).Errorf("replay like event failed, event_id=%s, err=%v", record.EventID, err)
			nextRetryAt := nextLikeEventRetryTime(time.Now(), record.RetryCount+1)
			if markErr := repo.MarkRetry(record.EventID, nextRetryAt, err.Error()); markErr != nil {
				logx.WithContext(ctx).Errorf("mark replay retry failed, event_id=%s, err=%v", record.EventID, markErr)
			}
			continue
		}

		if err := repo.MarkSent(record.EventID, time.Now()); err != nil {
			logx.WithContext(ctx).Errorf("mark replay sent failed, event_id=%s, err=%v", record.EventID, err)
		}
	}
}

func (p *LikeProducer) dispatchPayload(ctx context.Context, payload string) error {
	var lastErr error
	for i := 0; i < p.maxRetries; i++ {
		if err := p.sendRaw(ctx, payload); err == nil {
			return nil
		} else {
			lastErr = err
			logx.WithContext(ctx).Errorf("send like event failed, retry %d/%d, err=%v", i+1, p.maxRetries, err)
			time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("send like event failed")
}

func (p *LikeProducer) sendRaw(ctx context.Context, payload string) error {
	return p.pusher.Push(ctx, payload)
}

func nextLikeEventRetryTime(now time.Time, retryCount int) time.Time {
	if retryCount < 1 {
		retryCount = 1
	}

	backoff := time.Second * time.Duration(1<<minLikeEventRetryShift(retryCount-1))
	maxBackoff := 5 * time.Minute
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return now.Add(backoff)
}

func minLikeEventRetryShift(v int) int {
	if v < 0 {
		return 0
	}
	if v > 8 {
		return 8
	}
	return v
}

func (r *LikeOutboxRelay) Start() {
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

func (r *LikeOutboxRelay) Stop() {
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
