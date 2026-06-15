package consumer

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

type dailyAggregator interface {
	Aggregate(ctx context.Context, event track.Event) error
}

type RecommendTrackConsumer struct {
	aggregator dailyAggregator
	logx.Logger
}

func NewRecommendTrackConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendTrackConsumer {
	var aggregator dailyAggregator
	if svcCtx != nil {
		aggregator = svcCtx.RecommendDailyAggregator
	}
	return newRecommendTrackConsumerForTest(ctx, aggregator)
}

func newRecommendTrackConsumerForTest(ctx context.Context, aggregator dailyAggregator) *RecommendTrackConsumer {
	return &RecommendTrackConsumer{
		aggregator: aggregator,
		Logger:     logx.WithContext(ctx),
	}
}

func (c *RecommendTrackConsumer) Consume(ctx context.Context, _, val string) error {
	var event track.Event
	if err := json.Unmarshal([]byte(val), &event); err != nil {
		logc.Errorf(ctx, "parse recommend track event failed, err=%v", err)
		return err
	}

	if c == nil || c.aggregator == nil {
		return nil
	}
	if err := c.aggregator.Aggregate(ctx, event); err != nil {
		logc.Errorf(ctx, "aggregate recommend track event failed, event_id=%s, err=%v", event.EventID, err)
		return err
	}
	return nil
}
