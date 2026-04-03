package producer

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/interaction/internal/mq/event"
)

type LikeProducer struct {
	pusher     *kq.Pusher
	maxRetries int
}

type EventProducer interface {
	SendLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string)
	SendCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string)
}

func NewLikeProducer(pusher *kq.Pusher, maxRetries int) *LikeProducer {
	return &LikeProducer{
		pusher:     pusher,
		maxRetries: maxRetries,
	}
}

func (p *LikeProducer) SendLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
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

	p.sendEventWithRetry(ctx, likeEvent)
}

func (p *LikeProducer) SendCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
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

	p.sendEventWithRetry(ctx, likeEvent)
}

func (p *LikeProducer) sendEventWithRetry(ctx context.Context, evt *event.LikeEvent) {
	var lastErr error
	for i := 0; i < p.maxRetries; i++ {
		if err := p.sendEvent(ctx, evt); err == nil {
			return
		} else {
			lastErr = err
			logx.WithContext(ctx).Errorf("send like event failed, retry %d/%d, err=%v", i+1, p.maxRetries, err)
			time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
		}
	}

	if lastErr != nil {
		logx.WithContext(ctx).Errorf("send like event failed after retries, event_id=%s, err=%v", evt.EventID, lastErr)
	}
}

func (p *LikeProducer) sendEvent(ctx context.Context, evt *event.LikeEvent) error {
	body, err := evt.Marshal()
	if err != nil {
		return err
	}

	if err = p.pusher.Push(ctx, string(body)); err != nil {
		return err
	}
	return nil
}
