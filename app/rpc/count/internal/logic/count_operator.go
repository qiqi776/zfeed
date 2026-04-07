package logic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
)

type CountOperator struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewCountOperator(ctx context.Context, svcCtx *svc.ServiceContext) *CountOperator {
	return &CountOperator{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (o *CountOperator) ApplyDelta(
	bizType count.BizType,
	targetType count.TargetType,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) error {
	return o.ApplyDeltaWithRepo(o.countRepo, bizType, targetType, targetID, ownerID, delta, updatedAt)
}

func (o *CountOperator) ApplyDeltaWithRepo(
	repo repositories.CountValueRepository,
	bizType count.BizType,
	targetType count.TargetType,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) error {
	if bizType == count.BizType_BIZ_TYPE_UNKNOWN ||
		targetType == count.TargetType_TARGET_TYPE_UNKNOWN ||
		targetID <= 0 || delta == 0 {
		return nil
	}

	if _, err := repo.ApplyDelta(int32(bizType), int32(targetType), targetID, ownerID, delta, updatedAt); err != nil {
		return err
	}
	o.InvalidateCountCache(bizType, targetType, targetID)
	if targetType == count.TargetType_CONTENT && ownerID > 0 {
		o.InvalidateUserProfileCountsCache(ownerID)
	}
	if targetType == count.TargetType_USER && targetID > 0 {
		o.InvalidateUserProfileCountsCache(targetID)
	}
	return nil
}

func (o *CountOperator) InvalidateCountCache(bizType count.BizType, targetType count.TargetType, targetID int64) {
	if bizType == count.BizType_BIZ_TYPE_UNKNOWN ||
		targetType == count.TargetType_TARGET_TYPE_UNKNOWN ||
		targetID <= 0 {
		return
	}
	cacheKey := buildCountValueCacheKey(bizType, targetType, targetID)
	if _, err := o.svcCtx.Redis.DelCtx(o.ctx, cacheKey); err != nil {
		o.Errorf("delete count cache failed, key=%s, err=%v", cacheKey, err)
	}
}

func (o *CountOperator) InvalidateUserProfileCountsCache(userID int64) {
	if userID <= 0 {
		return
	}
	cacheKey := buildUserProfileCountsCacheKey(userID)
	if _, err := o.svcCtx.Redis.DelCtx(o.ctx, cacheKey); err != nil {
		o.Errorf("delete user profile counts cache failed, key=%s, err=%v", cacheKey, err)
	}
}

func defaultUpdatedAt(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now()
	}
	return ts
}
