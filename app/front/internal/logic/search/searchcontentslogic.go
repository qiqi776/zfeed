package search

import (
	"context"
	"strings"
	"time"

	"zfeed/app/front/internal/svc"
	"zfeed/app/front/internal/types"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	contentStatusPublished  = 30
	contentVisibilityPublic = 10
	maxSearchPageSize       = 20
)

type SearchContentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type searchContentRow struct {
	ContentID    int64      `gorm:"column:content_id"`
	ContentType  int32      `gorm:"column:content_type"`
	AuthorID     int64      `gorm:"column:author_id"`
	AuthorName   string     `gorm:"column:author_name"`
	AuthorAvatar string     `gorm:"column:author_avatar"`
	Title        string     `gorm:"column:title"`
	CoverURL     string     `gorm:"column:cover_url"`
	PublishedAt  *time.Time `gorm:"column:published_at"`
}

func NewSearchContentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchContentsLogic {
	return &SearchContentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchContentsLogic) SearchContents(req *types.SearchContentsReq) (*types.SearchContentsRes, error) {
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
		Table("zfeed_content AS c").
		Select(`
			c.id AS content_id,
			c.content_type AS content_type,
			c.user_id AS author_id,
			COALESCE(u.nickname, '') AS author_name,
			COALESCE(u.avatar, '') AS author_avatar,
			COALESCE(a.title, v.title, '') AS title,
			COALESCE(a.cover, v.cover_url, '') AS cover_url,
			c.published_at AS published_at
		`).
		Joins("LEFT JOIN zfeed_article AS a ON a.content_id = c.id AND a.is_deleted = 0").
		Joins("LEFT JOIN zfeed_video AS v ON v.content_id = c.id AND v.is_deleted = 0").
		Joins("LEFT JOIN zfeed_user AS u ON u.id = c.user_id AND u.is_deleted = 0").
		Where("c.status = ? AND c.visibility = ? AND c.is_deleted = 0", contentStatusPublished, contentVisibilityPublic).
		Where("(a.title LIKE ? OR v.title LIKE ? OR a.description LIKE ? OR v.description LIKE ?)", pattern, pattern, pattern, pattern)

	if cursor > 0 {
		dbQuery = dbQuery.Where("c.id < ?", cursor)
	}

	rows := make([]searchContentRow, 0, pageSize+1)
	if err := dbQuery.Order("c.id DESC").Limit(pageSize + 1).Find(&rows).Error; err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("搜索内容失败"))
	}

	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}

	items := make([]types.SearchContentItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.SearchContentItem{
			ContentId:    row.ContentID,
			ContentType:  row.ContentType,
			AuthorId:     row.AuthorID,
			AuthorName:   row.AuthorName,
			AuthorAvatar: row.AuthorAvatar,
			Title:        row.Title,
			CoverUrl:     row.CoverURL,
			PublishedAt:  unixOrZero(row.PublishedAt),
		})
	}

	nextCursor := int64(0)
	if hasMore && len(items) > 0 {
		nextCursor = items[len(items)-1].ContentId
	}

	return &types.SearchContentsRes{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func unixOrZero(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}
