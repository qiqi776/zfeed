package consumer

import (
	"context"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"

	"zfeed/app/rpc/interaction/internal/config"
	"zfeed/app/rpc/interaction/internal/svc"
)

func Consumers(c config.Config, ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	return []service.Service{
		kq.MustNewQueue(c.KqConsumerConf, NewLikeConsumer(ctx, svcCtx)),
	}
}
