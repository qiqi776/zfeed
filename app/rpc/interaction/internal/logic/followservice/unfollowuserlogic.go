package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/do"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnfollowUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewUnfollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnfollowUserLogic {
	return &UnfollowUserLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *UnfollowUserLogic) UnfollowUser(in *interaction.UnfollowUserReq) (*interaction.UnfollowUserRes, error) {
	if in == nil || in.GetUserId() <= 0 || in.GetFollowUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetUserId() == in.GetFollowUserId() {
		return nil, errorx.NewBadRequest("不能取关自己")
	}

	err := l.followRepo.Upsert(&do.FollowDO{
		UserID:       in.GetUserId(),
		FollowUserID: in.GetFollowUserId(),
		Status:       repositories.FollowStatusUnfollow,
		CreatedBy:    in.GetUserId(),
		UpdatedBy:    in.GetUserId(),
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消关注失败"))
	}

	return &interaction.UnfollowUserRes{IsFollowed: false}, nil
}
