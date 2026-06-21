package admin

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Update Content Status Logic ====================

type AdminUpdateContentStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminUpdateContentStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateContentStatusLogic {
	return &AdminUpdateContentStatusLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminUpdateContentStatusReq struct {
	ContentId int64 `json:"content_id"`
	Status    int32 `json:"status"` // 20=hidden, 30=published, 40=deleted
}

type AdminUpdateContentStatusRes struct {
	Affected bool `json:"affected"`
}

func (l *AdminUpdateContentStatusLogic) AdminUpdateContentStatus(req *AdminUpdateContentStatusReq) (*AdminUpdateContentStatusRes, error) {
	adminID, err := getAdminUserID(l.ctx)
	if err != nil {
		return nil, err
	}

	if req.ContentId <= 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	db, dbErr := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if dbErr != nil {
		return nil, dbErr
	}
	defer db.Close()

	var contentType int32
	err = l.applyContentStatus(db, req.ContentId, req.Status, &contentType)
	if err != nil {
		l.Logger.Infof("AdminUpdateContentStatus: content=%d status=%d admin=%d error=%v", req.ContentId, req.Status, adminID, err)
		return nil, err
	}

	l.Logger.Infof("AdminUpdateContentStatus: content=%d status=%d content_type=%d admin=%d OK", req.ContentId, req.Status, contentType, adminID)
	return &AdminUpdateContentStatusRes{Affected: true}, nil
}

// applyContentStatus performs the database update for a single content status change.
// It handles status 20 (hide), 30 (restore/publish), and 40 (soft delete).
func (l *AdminUpdateContentStatusLogic) applyContentStatus(db *sql.DB, contentID int64, status int32, contentType *int32) error {
	var row struct {
		ID          int64
		ContentType int32
		IsDeleted   int32
	}

	err := db.QueryRow(
		"SELECT id, content_type, is_deleted FROM zfeed_content WHERE id = ?",
		contentID,
	).Scan(&row.ID, &row.ContentType, &row.IsDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return errorx.NewNotFound("内容不存在")
		}
		return errorx.NewMsg("查询内容失败")
	}
	if row.IsDeleted != 0 {
		return errorx.NewNotFound("内容不存在")
	}
	*contentType = row.ContentType

	switch status {
	case 20: // hide
		_, err = db.Exec("UPDATE zfeed_content SET status = 20 WHERE id = ? AND is_deleted = 0", contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("隐藏内容失败"))
		}
	case 30: // restore / publish
		_, err = db.Exec("UPDATE zfeed_content SET status = 30 WHERE id = ? AND is_deleted = 0", contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("恢复内容失败"))
		}
	case 40: // soft delete
		_, err = db.Exec("UPDATE zfeed_content SET is_deleted = 1, status = 0 WHERE id = ?", contentID)
		if err != nil {
			return errorx.Wrap(l.ctx, err, errorx.NewMsg("删除内容失败"))
		}
		// Also mark the detail table
		switch row.ContentType {
		case 10:
			db.Exec("UPDATE zfeed_article SET is_deleted = 1 WHERE content_id = ?", contentID)
		case 20:
			db.Exec("UPDATE zfeed_video SET is_deleted = 1 WHERE content_id = ?", contentID)
		}
	default:
		return errorx.NewBadRequest("无效的状态值")
	}

	return nil
}

// ==================== Admin Batch Update Content Status ====================

type AdminBatchContentStatusReq struct {
	ContentIds []int64 `json:"content_ids"`
	Status     int32   `json:"status"`
}

type AdminBatchContentStatusRes struct {
	AffectedCount int32   `json:"affected_count"`
	FailedIds     []int64 `json:"failed_ids,omitempty"`
}

func (l *AdminUpdateContentStatusLogic) AdminBatchContentStatus(req *AdminBatchContentStatusReq) (*AdminBatchContentStatusRes, error) {
	adminID, err := getAdminUserID(l.ctx)
	if err != nil {
		return nil, err
	}

	if len(req.ContentIds) == 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	db, dbErr := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if dbErr != nil {
		return nil, dbErr
	}
	defer db.Close()

	var affected int32
	var failedIds []int64
	var contentType int32

	for _, cid := range req.ContentIds {
		if e := l.applyContentStatus(db, cid, req.Status, &contentType); e != nil {
			failedIds = append(failedIds, cid)
			l.Logger.Infof("AdminBatchContentStatus: content=%d status=%d admin=%d error=%v", cid, req.Status, adminID, e)
		} else {
			affected++
		}
	}

	l.Logger.Infof("AdminBatchContentStatus: status=%d total=%d affected=%d failed=%d admin=%d",
		req.Status, len(req.ContentIds), affected, len(failedIds), adminID)

	return &AdminBatchContentStatusRes{AffectedCount: affected, FailedIds: failedIds}, nil
}
