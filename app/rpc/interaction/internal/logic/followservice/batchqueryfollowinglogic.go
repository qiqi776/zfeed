package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchQueryFollowingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewBatchQueryFollowingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchQueryFollowingLogic {
	return &BatchQueryFollowingLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchQueryFollowingLogic) BatchQueryFollowing(in *interaction.BatchQueryFollowingReq) (*interaction.BatchQueryFollowingRes, error) {
	if in == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}
	if in.GetUserId() <= 0 || len(in.GetFollowUserIds()) == 0 {
		return &interaction.BatchQueryFollowingRes{
			Items: []*interaction.FollowingState{},
		}, nil
	}

	seen := make(map[int64]struct{}, len(in.GetFollowUserIds()))
	ids := make([]int64, 0, len(in.GetFollowUserIds()))
	for _, followUserID := range in.GetFollowUserIds() {
		if followUserID <= 0 {
			continue
		}
		if _, ok := seen[followUserID]; ok {
			continue
		}
		seen[followUserID] = struct{}{}
		ids = append(ids, followUserID)
	}
	if len(ids) == 0 {
		return &interaction.BatchQueryFollowingRes{
			Items: []*interaction.FollowingState{},
		}, nil
	}

	followingMap, err := l.followRepo.BatchFollowing(in.GetUserId(), ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注关系失败"))
	}

	items := make([]*interaction.FollowingState, 0, len(ids))
	for _, userID := range ids {
		items = append(items, &interaction.FollowingState{
			UserId:      userID,
			IsFollowing: followingMap[userID],
		})
	}

	return &interaction.BatchQueryFollowingRes{
		Items: items,
	}, nil
}
