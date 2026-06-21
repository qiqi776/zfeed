package admin

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Update User Status Logic ====================

type AdminUpdateUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminUpdateUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateUserStatusLogic {
	return &AdminUpdateUserStatusLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminUpdateUserStatusReq struct {
	UserId int64 `json:"user_id"`
	Status int32 `json:"status"` // 10=active, 20=disabled
}

type AdminUpdateUserStatusRes struct {
	Affected bool `json:"affected"`
}

func (l *AdminUpdateUserStatusLogic) AdminUpdateUserStatus(req *AdminUpdateUserStatusReq) (*AdminUpdateUserStatusRes, error) {
	adminID, err := getAdminUserID(l.ctx)
	if err != nil {
		return nil, err
	}
	_ = adminID // used for audit logging

	if req.UserId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	if req.Status != 10 && req.Status != 20 {
		return nil, errorx.NewBadRequest("无效的用户状态")
	}

	// Write status directly to MySQL
	db, dbErr := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if dbErr != nil {
		return nil, dbErr
	}
	defer db.Close()

	result, err := db.Exec("UPDATE zfeed_user SET status = ? WHERE id = ?", req.Status, req.UserId)
	if err != nil {
		l.Logger.Errorf("AdminUpdateUserStatus DB error: user=%d status=%d err=%v", req.UserId, req.Status, err)
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新用户状态失败"))
	}

	rowsAffected, _ := result.RowsAffected()
	l.Logger.Infof("AdminUpdateUserStatus: admin=%d user=%d new_status=%d rows_affected=%d", adminID, req.UserId, req.Status, rowsAffected)

	return &AdminUpdateUserStatusRes{Affected: rowsAffected > 0}, nil
}
