package consumer

import (
	"context"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/mq/event"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
)

type LikeConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	consumerName string
}

func NewLikeConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *LikeConsumer {
	return &LikeConsumer{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		consumerName: "interaction.like_consumer",
	}
}

func (c *LikeConsumer) Consume(ctx context.Context, key, val string) error {
	logc.Infof(ctx, "start consume like event: %s", val)

	likeEvent, err := event.UnmarshalLikeEvent(val)
	if err != nil {
		logc.Errorf(ctx, "unmarshal like event failed, err=%v", err)
		return err
	}

	status := repositories.LikeStatusCancel
	if likeEvent.EventType == event.EventTypeLike {
		status = repositories.LikeStatusLike
	}

	return c.svcCtx.MysqlDb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dedupRepo := repositories.NewMqConsumeDedupRepository(ctx, tx)
		inserted, err := dedupRepo.InsertIfAbsent(c.consumerName, likeEvent.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}

		likeDO := &do.LikeDO{
			UserID:        likeEvent.UserID,
			ContentID:     likeEvent.ContentID,
			ContentUserID: likeEvent.ContentUserID,
			Status:        status,
			LastEventTs:   likeEvent.Timestamp,
			CreatedBy:     likeEvent.UserID,
			UpdatedBy:     likeEvent.UserID,
		}

		if err = repositories.NewLikeRepository(ctx, tx).Upsert(likeDO); err != nil {
			logc.Errorf(ctx, "upsert like event failed, event_id=%s, err=%v", likeEvent.EventID, err)
			return err
		}
		return nil
	})
}
