package logic

import (
	"context"
	"strconv"

	"zfeed/app/rpc/count/count"
	redisconsts "zfeed/app/rpc/count/internal/common/consts/redis"
	"zfeed/app/rpc/count/internal/repositories"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	countRepo repositories.CountValueRepository
}

func NewGetCountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCountLogic {
	return &GetCountLogic{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		countRepo: repositories.NewCountValueRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetCountLogic) GetCount(in *count.GetCountReq) (*count.GetCountRes, error) {
	if in == nil || in.GetTargetId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetBizType() == count.BizType_BIZ_TYPE_UNKNOWN || in.GetTargetType() == count.TargetType_TARGET_TYPE_UNKNOWN {
		return nil, errorx.NewMsg("参数错误")
	}

	cacheKey := buildCountValueCacheKey(in.GetBizType(), in.GetTargetType(), in.GetTargetId())
	cacheStr, err := l.svcCtx.Redis.GetCtx(l.ctx, cacheKey)
	if err == nil && cacheStr != "" {
		if value, parseErr := strconv.ParseInt(cacheStr, 10, 64); parseErr == nil {
			return &count.GetCountRes{Value: value}, nil
		}
	}

	row, err := l.countRepo.Get(int32(in.GetBizType()), int32(in.GetTargetType()), in.GetTargetId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询计数失败"))
	}

	value := int64(0)
	if row != nil {
		value = row.Value
	}
	if err := l.svcCtx.Redis.SetexCtx(
		l.ctx,
		cacheKey,
		strconv.FormatInt(value, 10),
		countCacheExpireSecondsWithJitter(redisconsts.RedisCountValueExpireSeconds),
	); err != nil {
		l.Errorf("set count cache failed, key=%s, err=%v", cacheKey, err)
	}

	return &count.GetCountRes{Value: value}, nil
}
