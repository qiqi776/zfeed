package consumer

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	gzredis "github.com/zeromicro/go-zero/core/stores/redis"

	contentconfig "zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/content/internal/svc"
)

type dailyAggregator interface {
	Aggregate(ctx context.Context, event track.Event) error
}

type profileUpdater interface {
	Apply(ctx context.Context, event track.Event) error
}

type RecommendTrackConsumer struct {
	aggregator     dailyAggregator
	profileUpdater profileUpdater
	logx.Logger
}

func NewRecommendTrackConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendTrackConsumer {
	var aggregator dailyAggregator
	var updater profileUpdater
	if svcCtx != nil {
		aggregator = svcCtx.RecommendDailyAggregator
		if svcCtx.Redis != nil {
			updater = redisProfileUpdater{
				rds: svcCtx.Redis,
				cfg: svcCtx.Config.Recommend,
			}
		}
	}
	return newRecommendTrackConsumerWithProfileForTest(ctx, aggregator, updater)
}

func newRecommendTrackConsumerForTest(ctx context.Context, aggregator dailyAggregator) *RecommendTrackConsumer {
	return newRecommendTrackConsumerWithProfileForTest(ctx, aggregator, nil)
}

func newRecommendTrackConsumerWithProfileForTest(
	ctx context.Context,
	aggregator dailyAggregator,
	updater profileUpdater,
) *RecommendTrackConsumer {
	return &RecommendTrackConsumer{
		aggregator:     aggregator,
		profileUpdater: updater,
		Logger:         logx.WithContext(ctx),
	}
}

func (c *RecommendTrackConsumer) Consume(ctx context.Context, _, val string) error {
	var event track.Event
	if err := json.Unmarshal([]byte(val), &event); err != nil {
		logc.Errorf(ctx, "parse recommend track event failed, err=%v", err)
		return err
	}

	if c != nil && c.profileUpdater != nil {
		if err := c.profileUpdater.Apply(ctx, event); err != nil {
			logc.Errorf(ctx, "apply recommend profile event failed, event_id=%s, err=%v", event.EventID, err)
			return err
		}
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

type redisProfileUpdater struct {
	rds *gzredis.Redis
	cfg contentconfig.RecommendConfig
}

func (u redisProfileUpdater) Apply(ctx context.Context, event track.Event) error {
	action, ok := recommend.ProfileActionForTrackEvent(event.EventType, event.DwellMs)
	if !ok {
		return nil
	}

	return recommend.ApplyProfileEvent(ctx, u.rds, u.cfg, recommend.ProfileEvent{
		EventID:   event.EventID,
		EventType: action,
		UserID:    event.UserID,
		ContentID: event.ContentID,
	})
}
