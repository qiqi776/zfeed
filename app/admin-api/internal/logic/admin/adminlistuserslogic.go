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

// ==================== Admin List Users Logic ====================

type AdminListUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsersLogic {
	return &AdminListUsersLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminUserItem struct {
	UserId    int64  `json:"user_id"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Mobile    string `json:"mobile"`
	Status    int32  `json:"status"`
	Role      int32  `json:"role"`
	CreatedAt int64  `json:"created_at"`
}

type AdminListUsersReq struct {
	Query    string `json:"query"`
	Page     int32  `json:"page"`
	PageSize int32  `json:"page_size"`
	Status   *int32 `json:"status,optional,omitempty"`
	Role     *int32 `json:"role,optional,omitempty"`
}

type AdminListUsersRes struct {
	Users      []AdminUserItem `json:"users"`
	TotalCount int64           `json:"total_count"`
}

func (l *AdminListUsersLogic) AdminListUsers(req *AdminListUsersReq) (*AdminListUsersRes, error) {
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

	// Build query
	where := "WHERE is_deleted = 0"
	args := make([]interface{}, 0)
	if req.Query != "" {
		where += " AND (nickname LIKE ? OR mobile LIKE ?)"
		like := "%" + req.Query + "%"
		args = append(args, like, like)
	}
	if req.Status != nil {
		where += " AND status = ?"
		args = append(args, *req.Status)
	}
	if req.Role != nil {
		where += " AND role = ?"
		args = append(args, *req.Role)
	}

	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM zfeed_user %s", where)
	db.QueryRow(countQuery, args...).Scan(&total)

	dataQuery := fmt.Sprintf("SELECT id, nickname, avatar, mobile, status, role, UNIX_TIMESTAMP(created_at) FROM zfeed_user %s ORDER BY id DESC LIMIT ? OFFSET ?", where)
	dataArgs := append(args, pageSize, offset)
	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户列表失败"))
	}
	defer rows.Close()

	users := make([]AdminUserItem, 0)
	for rows.Next() {
		var u AdminUserItem
		var mobile, avatar sql.NullString
		if err := rows.Scan(&u.UserId, &u.Nickname, &avatar, &mobile, &u.Status, &u.Role, &u.CreatedAt); err != nil {
			continue
		}
		u.Mobile = mobile.String
		u.Avatar = avatar.String
		users = append(users, u)
	}

	return &AdminListUsersRes{
		Users:      users,
		TotalCount: total,
	}, nil
}
