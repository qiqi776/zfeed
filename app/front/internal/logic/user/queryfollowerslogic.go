package user

import (
	"context"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	followStatusActive = 10
	maxFollowersPage   = 50
)

type QueryFollowersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type followerRelationRow struct {
	UserID int64 `gorm:"column:user_id"`
}

type followerUserRow struct {
	ID       int64  `gorm:"column:id"`
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
	Bio      string `gorm:"column:bio"`
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

	pageSize := int(*req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxFollowersPage {
		pageSize = maxFollowersPage
	}

	cursor := int64(0)
	if req.Cursor != nil && *req.Cursor > 0 {
		cursor = *req.Cursor
	}

	rows, err := l.queryFollowerRelations(*req.UserId, cursor, pageSize+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}
	if len(rows) == 0 {
		return &types.QueryFollowersRes{
			Items:      []types.FollowerItem{},
			NextCursor: 0,
			HasMore:    false,
		}, nil
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	userIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.UserID <= 0 {
			continue
		}
		userIDs = append(userIDs, row.UserID)
	}

	userMap, err := l.queryFollowerUsers(userIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	followingMap, err := l.queryViewerFollowingMap(viewerID, userIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询粉丝列表失败"))
	}

	items := make([]types.FollowerItem, 0, len(userIDs))
	for _, userID := range userIDs {
		row, ok := userMap[userID]
		if !ok {
			continue
		}
		items = append(items, types.FollowerItem{
			UserId:      row.ID,
			Nickname:    row.Nickname,
			Avatar:      row.Avatar,
			Bio:         row.Bio,
			IsFollowing: followingMap[userID],
		})
	}

	nextCursor := int64(0)
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].UserId
	}

	return &types.QueryFollowersRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (l *QueryFollowersLogic) queryFollowerRelations(userID, cursor int64, limit int) ([]followerRelationRow, error) {
	query := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_follow").
		Select("user_id").
		Where("follow_user_id = ? AND status = ? AND is_deleted = 0", userID, followStatusActive)

	if cursor > 0 {
		query = query.Where("user_id < ?", cursor)
	}

	rows := make([]followerRelationRow, 0, limit)
	if err := query.Order("user_id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (l *QueryFollowersLogic) queryFollowerUsers(userIDs []int64) (map[int64]followerUserRow, error) {
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

func (l *QueryFollowersLogic) queryViewerFollowingMap(viewerID int64, userIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(userIDs))
	if viewerID <= 0 || len(userIDs) == 0 {
		return result, nil
	}

	type row struct {
		FollowUserID int64 `gorm:"column:follow_user_id"`
	}

	rows := make([]row, 0, len(userIDs))
	if err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_follow").
		Select("follow_user_id").
		Where("user_id = ? AND follow_user_id IN ? AND status = ? AND is_deleted = 0", viewerID, userIDs, followStatusActive).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, item := range rows {
		result[item.FollowUserID] = true
	}
	return result, nil
}
