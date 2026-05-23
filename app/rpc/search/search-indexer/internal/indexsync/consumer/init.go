package consumer

import (
	"context"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/search-indexer/internal/indexsvc"
)

func Consumers(c indexconfig.Config, ctx context.Context, svcCtx *indexsvc.ServiceContext) []service.Service {
	if c.KqConsumerConf.Topic == "" {
		return nil
	}
	return []service.Service{
		kq.MustNewQueue(c.KqConsumerConf, NewCanalSearchConsumer(ctx, svcCtx)),
	}
}
