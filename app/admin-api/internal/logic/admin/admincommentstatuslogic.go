package admin

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Update Comment Status Logic ====================

type AdminUpdateCommentStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminUpdateCommentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateCommentStatusLogic {
	return &AdminUpdateCommentStatusLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminUpdateCommentStatusReq struct {
	CommentId int64 `json:"comment_id"`
	Status    int32 `json:"status"` // 10=active, 20=hidden, 40=deleted
}

type AdminUpdateCommentStatusRes struct {
	Affected bool `json:"affected"`
}

func (l *AdminUpdateCommentStatusLogic) AdminUpdateCommentStatus(req *AdminUpdateCommentStatusReq) (*AdminUpdateCommentStatusRes, error) {
	adminID, err := getAdminUserID(l.ctx)
	if err != nil {
		return nil, err
	}
	_ = adminID // used for audit logging

	if req.CommentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	db, dbErr := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if dbErr != nil {
		return nil, dbErr
	}
	defer db.Close()

	switch req.Status {
	case 20: // hide comment — mark status on zfeed_comment
		result, err := db.Exec("UPDATE zfeed_comment SET status = 20 WHERE id = ? AND is_deleted = 0", req.CommentId)
		if err != nil {
			l.Logger.Errorf("AdminUpdateCommentStatus hide: comment=%d admin=%d err=%v", req.CommentId, adminID, err)
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("隐藏评论失败"))
		}
		rows, _ := result.RowsAffected()
		l.Logger.Infof("AdminUpdateCommentStatus: comment=%d status=20(hidden) admin=%d rows=%d", req.CommentId, adminID, rows)
		return &AdminUpdateCommentStatusRes{Affected: rows > 0}, nil
	case 10: // restore comment
		result, err := db.Exec("UPDATE zfeed_comment SET status = 10 WHERE id = ? AND is_deleted = 0", req.CommentId)
		if err != nil {
			l.Logger.Errorf("AdminUpdateCommentStatus restore: comment=%d admin=%d err=%v", req.CommentId, adminID, err)
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("恢复评论失败"))
		}
		rows, _ := result.RowsAffected()
		l.Logger.Infof("AdminUpdateCommentStatus: comment=%d status=10(active) admin=%d rows=%d", req.CommentId, adminID, rows)
		return &AdminUpdateCommentStatusRes{Affected: rows > 0}, nil
	case 40: // soft delete comment
		result, err := db.Exec("UPDATE zfeed_comment SET is_deleted = 1, status = 0 WHERE id = ?", req.CommentId)
		if err != nil {
			l.Logger.Errorf("AdminUpdateCommentStatus delete: comment=%d admin=%d err=%v", req.CommentId, adminID, err)
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除评论失败"))
		}
		rows, _ := result.RowsAffected()
		l.Logger.Infof("AdminUpdateCommentStatus: comment=%d status=40(deleted) admin=%d rows=%d", req.CommentId, adminID, rows)
		return &AdminUpdateCommentStatusRes{Affected: rows > 0}, nil
	default:
		return nil, errorx.NewBadRequest("无效的状态值")
	}
}
