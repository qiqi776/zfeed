package user

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	followservicepb "zfeed/app/rpc/interaction/client/followservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryFollowersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryFollowersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFollowersLogic {
	return &QueryFollowersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryFollowersLogic) QueryFollowers(req *types.QueryFollowersReq) (*types.QueryFollowersRes, error) {
	if req == nil || req.UserId == nil || *req.UserId <= 0 || req.PageSize == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if l.svcCtx == nil || l.svcCtx.FollowRpc == nil {
		return nil, errorx.NewMsg("查询粉丝列表失败")
	}

	cursor := int64(0)
	if req.Cursor != nil && *req.Cursor > 0 {
		cursor = *req.Cursor
	}

	var viewerID *int64
	if value := utils.GetContextUserIdWithDefault(l.ctx); value > 0 {
		viewerID = &value
	}

	rpcResp, err := l.svcCtx.FollowRpc.ListFollowers(l.ctx, &followservicepb.ListFollowersReq{
		UserId:   *req.UserId,
		Cursor:   cursor,
		PageSize: *req.PageSize,
		ViewerId: viewerID,
	})
	if err != nil {
		return nil, err
	}

	items := make([]types.FollowerItem, 0, len(rpcResp.GetItems()))
	for _, item := range rpcResp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, types.FollowerItem{
			UserId:      item.GetUserId(),
			Nickname:    item.GetNickname(),
			Avatar:      item.GetAvatar(),
			Bio:         item.GetBio(),
			IsFollowing: item.GetIsFollowing(),
		})
	}

	return &types.QueryFollowersRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}
