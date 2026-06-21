package admin

import (
	"context"
	"database/sql"

	"github.com/zeromicro/go-zero/core/logx"

	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/errorx"

	_ "github.com/go-sql-driver/mysql"
)

// ==================== Admin Get Settings Logic ====================

type AdminGetSettingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminGetSettingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetSettingsLogic {
	return &AdminGetSettingsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type AdminSettingsRes struct {
	Settings map[string]string `json:"settings"`
}

// defaultSettings defines the fallback values when no settings exist in DB.
var defaultSettings = map[string]string{
	"enable_recommend":    "true",
	"enable_search_index": "true",
	"content_review_mode": "post",
}

func (l *AdminGetSettingsLogic) AdminGetSettings() (*AdminSettingsRes, error) {
	db, err := sql.Open("mysql", l.svcCtx.Config.MySQL.DataSource)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Ensure the settings table exists (idempotent)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS zfeed_admin_setting (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		setting_key VARCHAR(128) NOT NULL UNIQUE,
		setting_value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)

	// Seed defaults if table is empty
	var count int64
	db.QueryRow("SELECT COUNT(*) FROM zfeed_admin_setting").Scan(&count)
	if count == 0 {
		for k, v := range defaultSettings {
			db.Exec("INSERT IGNORE INTO zfeed_admin_setting (setting_key, setting_value) VALUES (?, ?)", k, v)
		}
	}

	// Read all settings
	rows, err := db.Query("SELECT setting_key, setting_value FROM zfeed_admin_setting")
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("读取设置失败"))
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		settings[key] = value
	}

	// Merge defaults for any keys not yet in DB
	for k, v := range defaultSettings {
		if _, ok := settings[k]; !ok {
			settings[k] = v
		}
	}

	return &AdminSettingsRes{Settings: settings}, nil
}
