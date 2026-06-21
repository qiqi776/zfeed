package admin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin List Contents Logic ====================

type AdminListContentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListContentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListContentsLogic {
	return &AdminListContentsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminContentItem struct {
	ContentId    int64  `json:"content_id"`
	ContentType  int32  `json:"content_type"`
	AuthorId     int64  `json:"author_id"`
	AuthorName   string `json:"author_name"`
	Title        string `json:"title"`
	CoverUrl     string `json:"cover_url"`
	Status       int32  `json:"status"`
	PublishedAt  int64  `json:"published_at"`
	LikeCount    int64  `json:"like_count"`
	CommentCount int64  `json:"comment_count"`
}

type AdminListContentsReq struct {
	Query    string `json:"query"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Status   *int32 `json:"status,optional,omitempty"`
	UserId   *int64 `json:"user_id,optional,omitempty"`
}

type AdminListContentsRes struct {
	Contents   []AdminContentItem `json:"contents"`
	TotalCount int64              `json:"total_count"`
}

func (l *AdminListContentsLogic) AdminListContents(req *AdminListContentsReq) (*AdminListContentsRes, error) {
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	db, err := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	where := "WHERE c.is_deleted = 0"
	args := make([]interface{}, 0)
	if req.Query != "" {
		where += " AND (a.title LIKE ? OR a.description LIKE ? OR v.title LIKE ? OR v.description LIKE ?)"
		like := "%" + req.Query + "%"
		args = append(args, like, like, like, like)
	}
	if req.Status != nil {
		where += " AND c.status = ?"
		args = append(args, *req.Status)
	}
	if req.UserId != nil && *req.UserId > 0 {
		where += " AND c.user_id = ?"
		args = append(args, *req.UserId)
	}

	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM zfeed_content c
		LEFT JOIN zfeed_article a ON a.content_id = c.id
		LEFT JOIN zfeed_video v ON v.content_id = c.id
		%s`, where)
	db.QueryRow(countQuery, args...).Scan(&total)

	dataQuery := fmt.Sprintf(`
		SELECT c.id, c.content_type, c.user_id, COALESCE(u.nickname, ''),
		       COALESCE(a.title, v.title, ''), COALESCE(a.cover, v.cover_url, ''),
		       c.status, UNIX_TIMESTAMP(c.created_at),
		       c.like_count, c.comment_count
		FROM zfeed_content c
		LEFT JOIN zfeed_article a ON a.content_id = c.id AND c.content_type = 10
		LEFT JOIN zfeed_video v ON v.content_id = c.id AND c.content_type = 20
		LEFT JOIN zfeed_user u ON u.id = c.user_id
		%s
		ORDER BY c.id DESC LIMIT ? OFFSET ?`, where)
	dataArgs := append(args, pageSize, offset)
	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询内容列表失败"))
	}
	defer rows.Close()

	contents := make([]AdminContentItem, 0)
	for rows.Next() {
		var c AdminContentItem
		var authorName, coverUrl sql.NullString
		if err := rows.Scan(&c.ContentId, &c.ContentType, &c.AuthorId, &authorName,
			&c.Title, &coverUrl, &c.Status, &c.PublishedAt, &c.LikeCount, &c.CommentCount); err != nil {
			continue
		}
		c.AuthorName = authorName.String
		c.CoverUrl = coverUrl.String
		contents = append(contents, c)
	}

	return &AdminListContentsRes{
		Contents:   contents,
		TotalCount: total,
	}, nil
}
