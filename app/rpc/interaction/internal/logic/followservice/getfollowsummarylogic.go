package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFollowSummaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewGetFollowSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFollowSummaryLogic {
	return &GetFollowSummaryLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetFollowSummaryLogic) GetFollowSummary(in *interaction.GetFollowSummaryReq) (*interaction.GetFollowSummaryRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	followeeCount, err := l.followRepo.CountFollowees(in.GetUserId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注信息失败"))
	}
	followerCount, err := l.followRepo.CountFollowers(in.GetUserId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注信息失败"))
	}

	isFollowing := false
	if viewerID := in.GetViewerId(); viewerID > 0 {
		isFollowing, err = l.followRepo.IsFollowing(viewerID, in.GetUserId())
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注信息失败"))
		}
	}

	return &interaction.GetFollowSummaryRes{
		FolloweeCount: followeeCount,
		FollowerCount: followerCount,
		IsFollowing:   isFollowing,
	}, nil
}
