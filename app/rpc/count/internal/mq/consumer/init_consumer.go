package consumer

import (
	"context"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"

	"zfeed/app/rpc/count/internal/config"
	"zfeed/app/rpc/count/internal/svc"
)

func Consumers(c config.Config, ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	if c.KqConsumerConf.Topic == "" {
		return nil
	}
	return []service.Service{
		kq.MustNewQueue(c.KqConsumerConf, NewCanalCountConsumer(ctx, svcCtx)),
	}
}
