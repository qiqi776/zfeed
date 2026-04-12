package logic

import (
	"context"
	"time"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type IncLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	operator *CountOperator
}

func NewIncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *IncLogic {
	return &IncLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		operator: NewCountOperator(ctx, svcCtx),
	}
}

func (l *IncLogic) Inc(in *count.IncReq) (*count.IncRes, error) {
	if in == nil || in.GetTargetId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetBizType() == count.BizType_BIZ_TYPE_UNKNOWN || in.GetTargetType() == count.TargetType_TARGET_TYPE_UNKNOWN {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if err := l.operator.ApplyDelta(in.GetBizType(), in.GetTargetType(), in.GetTargetId(), 0, 1, time.Now()); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("计数更新失败"))
	}

	return &count.IncRes{}, nil
}
