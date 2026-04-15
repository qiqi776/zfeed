package followservicelogic

import (
	"context"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/repositories"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFollowersPage = 50

type ListFollowersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

type followerUserRow struct {
	ID       int64  `gorm:"column:id"`
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
	Bio      string `gorm:"column:bio"`
}

func NewListFollowersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFollowersLogic {
	return &ListFollowersLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *ListFollowersLogic) ListFollowers(in *interaction.ListFollowersReq) (*interaction.ListFollowersRes, error) {
	if in == nil || in.GetUserId() <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxFollowersPage {
		pageSize = maxFollowersPage
	}

	userIDs, err := l.followRepo.ListFollowersByCursor(in.GetUserId(), in.GetCursor(), pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}

	hasMore := len(userIDs) > pageSize
	if hasMore {
		userIDs = userIDs[:pageSize]
	}

	userMap, err := l.queryFollowerUsers(userIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}

	followingMap, err := l.followRepo.BatchFollowing(in.GetViewerId(), userIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}

	items := make([]*interaction.FollowerProfile, 0, len(userIDs))
	for _, userID := range userIDs {
		row, ok := userMap[userID]
		if !ok {
			continue
		}
		items = append(items, &interaction.FollowerProfile{
			UserId:      row.ID,
			Nickname:    row.Nickname,
			Avatar:      row.Avatar,
			Bio:         row.Bio,
			IsFollowing: followingMap[userID],
		})
	}

	nextCursor := int64(0)
	if hasMore && len(userIDs) > 0 {
		nextCursor = userIDs[len(userIDs)-1]
	}

	return &interaction.ListFollowersRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (l *ListFollowersLogic) queryFollowerUsers(userIDs []int64) (map[int64]followerUserRow, error) {
	result := make(map[int64]followerUserRow, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	rows := make([]followerUserRow, 0, len(userIDs))
	if err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_user").
		Select("id", "nickname", "avatar", "bio").
		Where("id IN ? AND is_deleted = 0", userIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}
