package search

import (
	"context"
	"strings"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"
	"zfeed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

const followActiveStatus = 10

type SearchUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type searchUserRow struct {
	UserID   int64  `gorm:"column:user_id"`
	Nickname string `gorm:"column:nickname"`
	Avatar   string `gorm:"column:avatar"`
	Bio      string `gorm:"column:bio"`
}

type followStateRow struct {
	FollowUserID int64 `gorm:"column:follow_user_id"`
}

func NewSearchUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchUsersLogic {
	return &SearchUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchUsersLogic) SearchUsers(req *types.SearchUsersReq) (*types.SearchUsersRes, error) {
	if req == nil || req.Query == nil || req.PageSize == nil {
		return nil, errorx.NewBadRequest("参数错误")
	}

	query := strings.TrimSpace(*req.Query)
	if query == "" {
		return nil, errorx.NewBadRequest("搜索词不能为空")
	}

	pageSize := int(*req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > maxSearchPageSize {
		pageSize = maxSearchPageSize
	}

	cursor := int64(0)
	if req.Cursor != nil && *req.Cursor > 0 {
		cursor = *req.Cursor
	}

	pattern := "%" + query + "%"
	dbQuery := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_user").
		Select("id AS user_id", "nickname", "avatar", "bio").
		Where("is_deleted = 0").
		Where("(nickname LIKE ? OR bio LIKE ? OR mobile LIKE ?)", pattern, pattern, pattern)

	if cursor > 0 {
		dbQuery = dbQuery.Where("id < ?", cursor)
	}

	rows := make([]searchUserRow, 0, pageSize+1)
	if err := dbQuery.Order("id DESC").Limit(pageSize + 1).Find(&rows).Error; err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	viewerID := utils.GetContextUserIdWithDefault(l.ctx)
	followingMap, err := l.queryFollowingMap(viewerID, rows)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索用户失败"))
	}

	items := make([]types.SearchUserItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.SearchUserItem{
			UserId:      row.UserID,
			Nickname:    row.Nickname,
			Avatar:      row.Avatar,
			Bio:         row.Bio,
			IsFollowing: followingMap[row.UserID],
		})
	}

	nextCursor := int64(0)
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].UserId
	}

	return &types.SearchUsersRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (l *SearchUsersLogic) queryFollowingMap(viewerID int64, rows []searchUserRow) (map[int64]bool, error) {
	result := make(map[int64]bool)
	if viewerID <= 0 || len(rows) == 0 {
		return result, nil
	}

	userIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.UserID)
	}

	followRows := make([]followStateRow, 0, len(userIDs))
	if err := l.svcCtx.MysqlDb.WithContext(l.ctx).
		Table("zfeed_follow").
		Select("follow_user_id").
		Where("user_id = ? AND follow_user_id IN ? AND status = ? AND is_deleted = 0", viewerID, userIDs, followActiveStatus).
		Find(&followRows).Error; err != nil {
		return nil, err
	}

	for _, row := range followRows {
		result[row.FollowUserID] = true
	}
	return result, nil
}
