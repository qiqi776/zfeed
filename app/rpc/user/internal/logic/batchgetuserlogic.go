package logic

import (
	"context"

	"zfeed/app/rpc/user/internal/repositories"
	"zfeed/app/rpc/user/internal/svc"
	"zfeed/app/rpc/user/user"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewBatchGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetUserLogic {
	return &BatchGetUserLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *BatchGetUserLogic) BatchGetUser(in *user.BatchGetUserReq) (*user.BatchGetUserRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if len(in.GetUserIds()) == 0 {
		return &user.BatchGetUserRes{
			Users: []*user.UserInfo{},
		}, nil
	}

	seen := make(map[int64]struct{}, len(in.GetUserIds()))
	ids := make([]int64, 0, len(in.GetUserIds()))
	for _, userID := range in.GetUserIds() {
		if userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		ids = append(ids, userID)
	}
	if len(ids) == 0 {
		return &user.BatchGetUserRes{
			Users: []*user.UserInfo{},
		}, nil
	}

	userMap, err := l.userRepo.BatchGetByIDs(ids)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("批量查询用户失败"))
	}

	users := make([]*user.UserInfo, 0, len(ids))
	for _, userID := range ids {
		userDO := userMap[userID]
		if userDO == nil {
			continue
		}

		users = append(users, &user.UserInfo{
			UserId:   userDO.ID,
			Username: userDO.Username,
			Mobile:   userDO.Mobile,
			Nickname: userDO.Nickname,
			Avatar:   userDO.Avatar,
			Bio:      userDO.Bio,
			Gender:   user.Gender(userDO.Gender),
			Status:   user.UserStatus(userDO.Status),
		})
	}

	return &user.BatchGetUserRes{
		Users: users,
	}, nil
}
