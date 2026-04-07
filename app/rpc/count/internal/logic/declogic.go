package logic

import (
	"context"
	"time"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type DecLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	operator *CountOperator
}

func NewDecLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DecLogic {
	return &DecLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		operator: NewCountOperator(ctx, svcCtx),
	}
}

func (l *DecLogic) Dec(in *count.DecReq) (*count.DecRes, error) {
	if in == nil || in.GetTargetId() <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.GetBizType() == count.BizType_BIZ_TYPE_UNKNOWN || in.GetTargetType() == count.TargetType_TARGET_TYPE_UNKNOWN {
		return nil, errorx.NewMsg("参数错误")
	}
	if err := l.operator.ApplyDelta(in.GetBizType(), in.GetTargetType(), in.GetTargetId(), 0, -1, time.Now()); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("计数更新失败"))
	}

	return &count.DecRes{}, nil
}
