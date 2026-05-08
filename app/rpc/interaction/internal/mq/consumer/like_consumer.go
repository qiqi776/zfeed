package consumer

import (
	"context"
	"strings"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/config"
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

func Consumers(c config.Config, ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	return []service.Service{
		kq.MustNewQueue(c.KqConsumerConf, NewLikeConsumer(ctx, svcCtx)),
	}
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

	sceneValue, ok := interaction.Scene_value[strings.TrimSpace(likeEvent.Scene)]
	if !ok || sceneValue == int32(interaction.Scene_SCENE_UNKNOWN) {
		logc.Errorf(ctx, "invalid like scene, event_id=%s, scene=%s", likeEvent.EventID, likeEvent.Scene)
		return nil
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
			Scene:         sceneValue,
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
