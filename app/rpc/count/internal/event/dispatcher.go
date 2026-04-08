package event

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

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
	countRepo repositories.CountValueRepository
	operator  *logic.CountOperator
	registry  *strategy.Registry
	consumer  string
}

var errEventAlreadyConsumed = errors.New("count event already consumed")

func NewDispatcher(ctx context.Context, svcCtx *svc.ServiceContext, consumerName string) *Dispatcher {
	return &Dispatcher{
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		dedupRepo: repositories.NewMqConsumeDedupRepository(ctx, svcCtx.MysqlDb),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
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

	updates := tableStrategy.ExtractUpdates(ctx, evt)
	if len(updates) == 0 {
		return 0, nil
	}

	updatedAt := evt.Timestamp
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	var pendingInvalidations []strategy.Update
	for attempt := 0; attempt < 3; attempt++ {
		pendingInvalidations = make([]strategy.Update, 0, len(updates))
		err := d.svcCtx.MysqlDb.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if evt.EventID != "" {
				inserted, err := d.dedupRepo.WithTx(tx).InsertIfAbsent(d.consumer, evt.EventID)
				if err != nil {
					return err
				}
				if !inserted {
					return errEventAlreadyConsumed
				}
			}

			countRepo := d.countRepo.WithTx(tx)
			for _, update := range updates {
				if err := d.operator.ApplyDeltaWithRepoNoCache(
					countRepo,
					update.BizType,
					update.TargetType,
					update.TargetID,
					update.OwnerID,
					update.Delta,
					updatedAt,
				); err != nil {
					return err
				}
				pendingInvalidations = append(pendingInvalidations, update)
			}
			return nil
		})
		if errors.Is(err, errEventAlreadyConsumed) {
			return 0, nil
		}
		if err == nil {
			break
		}
		if !isRetryableDispatchErr(err) || attempt == 2 {
			return 0, err
		}
		time.Sleep(time.Duration(attempt+1) * 20 * time.Millisecond)
	}

	for _, update := range pendingInvalidations {
		d.operator.InvalidateForUpdate(update.BizType, update.TargetType, update.TargetID, update.OwnerID)
	}
	return len(pendingInvalidations), nil
}

func isRetryableDispatchErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "deadlock found when trying to get lock") ||
		strings.Contains(msg, "lock wait timeout exceeded") ||
		strings.Contains(msg, "database is locked")
}
