package consumer

import (
	"context"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"

	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/svc"
)

func Consumers(c config.Config, ctx context.Context, svcCtx *svc.ServiceContext) []service.Service {
	configs := recommendConsumerConfigs(c)
	if len(configs) == 0 {
		return []service.Service{}
	}

	consumers := make([]service.Service, 0, len(configs))
	for _, cfg := range configs {
		consumers = append(consumers, kq.MustNewQueue(cfg, NewRecommendTrackConsumer(ctx, svcCtx)))
	}
	return consumers
}

func recommendConsumerConfigs(c config.Config) []kq.KqConf {
	configs := []kq.KqConf{}
	if c.KqConsumerConf.Topic != "" {
		configs = append(configs, c.KqConsumerConf)
	}
	if c.KqUserActionConsumerConf.Topic != "" {
		configs = append(configs, c.KqUserActionConsumerConf)
	}
	return configs
}
