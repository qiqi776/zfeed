package user

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	followservicepb "zfeed/app/rpc/interaction/client/followservice"
	userpb "zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"
)

type QueryFollowingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryFollowingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryFollowingsLogic {
	return &QueryFollowingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *QueryFollowingsLogic) QueryFollowings(req *types.QueryFollowingsReq) (*types.QueryFollowingsRes, error) {
	if invalidQueryFollowingsReq(req) {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if l.svcCtx == nil || l.svcCtx.FollowRpc == nil || l.svcCtx.UserRpc == nil {
		return nil, errorx.NewMsg("查询关注列表失败")
	}

	cursor := int64(0)
	if req.Cursor != nil && *req.Cursor > 0 {
		cursor = *req.Cursor
	}

	rpcResp, err := l.svcCtx.FollowRpc.ListFollowees(l.ctx, &followservicepb.ListFolloweesReq{
		UserId:   *req.UserId,
		Cursor:   cursor,
		PageSize: *req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	if rpcResp == nil {
		return nil, errorx.NewMsg("查询关注列表失败")
	}

	followingIDs := rpcResp.GetFollowUserIds()
	userResp, err := l.svcCtx.UserRpc.BatchGetUser(l.ctx, &userpb.BatchGetUserReq{UserIds: followingIDs})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
	}
	if userResp == nil {
		return nil, errorx.NewMsg("查询关注列表失败")
	}

	followingMap := map[int64]bool{}
	if viewerID := utils.GetContextUserIdWithDefault(l.ctx); viewerID > 0 && len(followingIDs) > 0 {
		stateResp, err := l.svcCtx.FollowRpc.BatchQueryFollowing(l.ctx, &followservicepb.BatchQueryFollowingReq{
			UserId:        viewerID,
			FollowUserIds: followingIDs,
		})
		if err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注列表失败"))
		}
		for _, item := range stateResp.GetItems() {
			if item == nil {
				continue
			}
			followingMap[item.GetUserId()] = item.GetIsFollowing()
		}
	}

	userMap := make(map[int64]*userpb.UserInfo, len(userResp.GetUsers()))
	for _, item := range userResp.GetUsers() {
		if item == nil || item.GetUserId() <= 0 {
			continue
		}
		userMap[item.GetUserId()] = item
	}

	items := make([]types.FollowerItem, 0, len(followingIDs))
	for _, userID := range followingIDs {
		item, ok := userMap[userID]
		if !ok {
			continue
		}
		items = append(items, types.FollowerItem{
			UserId:      item.GetUserId(),
			Nickname:    item.GetNickname(),
			Avatar:      item.GetAvatar(),
			Bio:         item.GetBio(),
			IsFollowing: followingMap[userID],
		})
	}

	return &types.QueryFollowingsRes{
		Items:      items,
		NextCursor: rpcResp.GetNextCursor(),
		HasMore:    rpcResp.GetHasMore(),
	}, nil
}

func invalidQueryFollowingsReq(req *types.QueryFollowingsReq) bool {
	if req == nil || req.UserId == nil || *req.UserId <= 0 || req.PageSize == nil {
		return true
	}
	if *req.PageSize == 0 || *req.PageSize > 50 {
		return true
	}
	return req.Cursor != nil && *req.Cursor < 0
}
