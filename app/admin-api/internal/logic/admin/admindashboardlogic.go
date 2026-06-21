package admin

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Dashboard Logic ====================

type AdminDashboardLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminDashboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminDashboardLogic {
	return &AdminDashboardLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminDashboardReq struct{}

type AdminDashboardRes struct {
	TotalUsers       int64 `json:"total_users"`
	TotalContents    int64 `json:"total_contents"`
	TotalComments    int64 `json:"total_comments"`
	TodayNewUsers    int64 `json:"today_new_users"`
	TodayNewContents int64 `json:"today_new_contents"`
	PendingReview    int64 `json:"pending_review"`
}

func (l *AdminDashboardLogic) AdminDashboard(req *AdminDashboardReq) (*AdminDashboardRes, error) {
	db, err := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var totalUsers, totalContents, totalComments, todayUsers, todayContents, pending int64
	today := time.Now().Format("2006-01-02")

	db.QueryRow("SELECT COUNT(*) FROM zfeed_user WHERE is_deleted = 0").Scan(&totalUsers)
	db.QueryRow("SELECT COUNT(*) FROM zfeed_content WHERE is_deleted = 0").Scan(&totalContents)
	db.QueryRow("SELECT COUNT(*) FROM zfeed_comment WHERE is_deleted = 0").Scan(&totalComments)
	db.QueryRow("SELECT COUNT(*) FROM zfeed_user WHERE is_deleted = 0 AND DATE(created_at) = ?", today).Scan(&todayUsers)
	db.QueryRow("SELECT COUNT(*) FROM zfeed_content WHERE is_deleted = 0 AND DATE(created_at) = ?", today).Scan(&todayContents)
	db.QueryRow("SELECT COUNT(*) FROM zfeed_content WHERE is_deleted = 0 AND status = 10").Scan(&pending)

	return &AdminDashboardRes{
		TotalUsers:       totalUsers,
		TotalContents:    totalContents,
		TotalComments:    totalComments,
		TodayNewUsers:    todayUsers,
		TodayNewContents: todayContents,
		PendingReview:    pending,
	}, nil
}
