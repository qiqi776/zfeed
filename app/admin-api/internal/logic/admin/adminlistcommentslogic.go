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

// ==================== Admin List Comments Logic ====================

type AdminListCommentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListCommentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListCommentsLogic {
	return &AdminListCommentsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminCommentItem struct {
	CommentId int64  `json:"comment_id"`
	ContentId int64  `json:"content_id"`
	UserId    int64  `json:"user_id"`
	UserName  string `json:"user_name"`
	Comment   string `json:"comment"`
	Status    int32  `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

type AdminListCommentsReq struct {
	Query     string `json:"query"`
	Page      int32  `json:"page"`
	PageSize  int32  `json:"page_size"`
	Status    *int32 `json:"status,optional,omitempty"`
	ContentId *int64 `json:"content_id,optional,omitempty"`
}

type AdminListCommentsRes struct {
	Comments   []AdminCommentItem `json:"comments"`
	TotalCount int64              `json:"total_count"`
}

func (l *AdminListCommentsLogic) AdminListComments(req *AdminListCommentsReq) (*AdminListCommentsRes, error) {
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
		where += " AND (c.comment LIKE ? OR u.nickname LIKE ?)"
		like := "%" + req.Query + "%"
		args = append(args, like, like)
	}
	if req.Status != nil {
		where += " AND c.status = ?"
		args = append(args, *req.Status)
	}
	if req.ContentId != nil && *req.ContentId > 0 {
		where += " AND c.content_id = ?"
		args = append(args, *req.ContentId)
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM zfeed_comment c LEFT JOIN zfeed_user u ON u.id = c.user_id %s", where)
	db.QueryRow(countQuery, args...).Scan(&total)

	dataQuery := fmt.Sprintf(`
		SELECT c.id, c.content_id, c.user_id, COALESCE(u.nickname, ''), c.comment, c.status, UNIX_TIMESTAMP(c.created_at)
		FROM zfeed_comment c
		LEFT JOIN zfeed_user u ON u.id = c.user_id
		%s
		ORDER BY c.id DESC LIMIT ? OFFSET ?`, where)
	dataArgs := append(args, pageSize, offset)
	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询评论列表失败"))
	}
	defer rows.Close()

	comments := make([]AdminCommentItem, 0)
	for rows.Next() {
		var item AdminCommentItem
		var contentId, userId sql.NullInt64
		if err := rows.Scan(&item.CommentId, &contentId, &userId, &item.UserName, &item.Comment, &item.Status, &item.CreatedAt); err != nil {
			continue
		}
		item.ContentId = contentId.Int64
		item.UserId = userId.Int64
		comments = append(comments, item)
	}

	return &AdminListCommentsRes{
		Comments:   comments,
		TotalCount: total,
	}, nil
}
