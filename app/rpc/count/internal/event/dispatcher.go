package event

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/count/internal/changeevent"
	"zfeed/app/rpc/count/internal/event/strategy"
	"zfeed/app/rpc/count/internal/logic"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
)

type Dispatcher struct {
	svcCtx       *svc.ServiceContext
	logx.Logger
	dedupRepo repositories.MqConsumeDedupRepository
	operator  *logic.CountOperator
	registry  *strategy.Registry
	consumer  string
}

func NewDispatcher(ctx context.Context, svcCtx *svc.ServiceContext, consumerName string) *Dispatcher {
	return &Dispatcher{
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		dedupRepo: repositories.NewMqConsumeDedupRepository(ctx, svcCtx.MysqlDb),
		operator:  logic.NewCountOperator(ctx, svcCtx),
		registry:  strategy.NewDefaultRegistry(),
		consumer:  consumerName,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, evt changeevent.ChangeEvent) (int, error) {
	tableStrategy, ok := d.registry.Get(evt.Table)
	if !ok {
		return 0, nil
	}
	if evt.EventID != "" {
		inserted, err := d.dedupRepo.InsertIfAbsent(d.consumer, evt.EventID)
		if err != nil {
			return 0, err
		}
		if !inserted {
			return 0, nil
		}
	}

	updates := tableStrategy.ExtractUpdates(ctx, evt)
	if len(updates) == 0 {
		return 0, nil
	}

	applied := 0
	updatedAt := evt.Timestamp
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	for _, update := range updates {
		if err := d.operator.ApplyDelta(
			update.BizType,
			update.TargetType,
			update.TargetID,
			update.OwnerID,
			update.Delta,
			updatedAt,
		); err != nil {
			return applied, err
		}
		applied++
	}
	return applied, nil
}
