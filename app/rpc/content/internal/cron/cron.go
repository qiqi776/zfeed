package cron

import (
	"context"

	"zfeed/app/rpc/content/internal/cron/hot_bucket_cleanup"
	"zfeed/app/rpc/content/internal/cron/hot_cold_rebuild"
	"zfeed/app/rpc/content/internal/cron/hot_fast_update"
	"zfeed/app/rpc/content/internal/cron/hot_snapshot_refresh"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/xxljob"
)

func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	hot_fast_update.Register(ctx, executor, svcCtx)
	hot_cold_rebuild.Register(ctx, executor, svcCtx)
	hot_snapshot_refresh.Register(ctx, executor, svcCtx)
	hot_bucket_cleanup.Register(ctx, executor, svcCtx)
}
