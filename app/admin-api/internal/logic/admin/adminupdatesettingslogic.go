package admin

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Update Settings Logic ====================

type AdminUpdateSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminUpdateSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminUpdateSettingsLogic {
	return &AdminUpdateSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminUpdateSettingsReq struct {
	Settings map[string]string `json:"settings"`
}

type AdminUpdateSettingsRes struct {
	Updated int32 `json:"updated"`
}

// allowedSettingKeys defines which keys are valid to update.
var allowedSettingKeys = map[string]bool{
	"enable_recommend":    true,
	"enable_search_index": true,
	"content_review_mode": true,
}

// validReviewModes defines valid values for content_review_mode.
var validReviewModes = map[string]bool{
	"post": true,
	"pre":  true,
}

func (l *AdminUpdateSettingsLogic) AdminUpdateSettings(req *AdminUpdateSettingsReq) (*AdminUpdateSettingsRes, error) {
	adminID, err := getAdminUserID(l.ctx)
	if err != nil {
		return nil, err
	}

	if len(req.Settings) == 0 {
		return nil, errorx.NewBadRequest("参数错误")
	}

	// Validate keys and values
	for k, v := range req.Settings {
		if !allowedSettingKeys[k] {
			return nil, errorx.NewBadRequest("无效的设置项: " + k)
		}
		if k == "content_review_mode" && !validReviewModes[v] {
			return nil, errorx.NewBadRequest("无效的审核模式，仅支持 post 或 pre")
		}
		if (k == "enable_recommend" || k == "enable_search_index") && v != "true" && v != "false" {
			return nil, errorx.NewBadRequest("开关设置仅支持 true 或 false")
		}
	}

	db, dbErr := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if dbErr != nil {
		return nil, dbErr
	}
	defer db.Close()

	// Ensure table exists
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS zfeed_admin_setting (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		setting_key VARCHAR(128) NOT NULL UNIQUE,
		setting_value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

	var updated int32
	for k, v := range req.Settings {
		result, err := db.Exec(
			"INSERT INTO zfeed_admin_setting (setting_key, setting_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)",
			k, v,
		)
		if err != nil {
			l.Logger.Errorf("AdminUpdateSettings: key=%s value=%s admin=%d err=%v", k, v, adminID, err)
			continue
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			updated++
		}
	}

	l.Logger.Infof("AdminUpdateSettings: updated=%d admin=%d", updated, adminID)
	return &AdminUpdateSettingsRes{Updated: updated}, nil
}
