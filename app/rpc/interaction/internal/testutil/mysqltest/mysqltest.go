package mysqltest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultMySQLHost = "127.0.0.1"
	defaultMySQLPort = "33306"
	defaultMySQLUser = "zfeed"
	defaultMySQLPass = "123456"
	defaultMySQLDB   = "zfeed"
	defaultMySQLLoc  = "Asia%2FShanghai"
)

var loadEnvOnce sync.Once

func Open() (*gorm.DB, error) {
	loadEnv()

	db, err := gorm.Open(mysql.Open(testDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return db, nil
}

func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func EnsureLikeTables(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}

	for _, ddl := range []string{
		createMqConsumeDedupTableDDL,
		createLikeTableDDL,
	} {
		if err := db.Exec(ddl).Error; err != nil {
			return err
		}
	}

	if err := ensureUniqueIndex(
		db,
		"zfeed_mq_consume_dedup",
		"uniq_consumer_event",
		"ALTER TABLE zfeed_mq_consume_dedup ADD UNIQUE KEY uniq_consumer_event (consumer, event_id)",
	); err != nil {
		return err
	}

	if err := ensureUniqueIndex(
		db,
		"zfeed_like",
		"uk_user_content",
		"ALTER TABLE zfeed_like ADD UNIQUE KEY uk_user_content (user_id, content_id)",
	); err != nil {
		return err
	}

	if err := ensureColumn(
		db,
		"zfeed_like",
		"last_event_ts",
		"ALTER TABLE zfeed_like ADD COLUMN last_event_ts BIGINT NOT NULL DEFAULT 0 AFTER status",
	); err != nil {
		return err
	}

	return nil
}

func CleanupLikeRowsByRange(db *gorm.DB, minID, maxID int64) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}

	return db.Exec(
		`DELETE FROM zfeed_like
WHERE (user_id BETWEEN ? AND ?)
   OR (content_id BETWEEN ? AND ?)
   OR (content_user_id BETWEEN ? AND ?)`,
		minID,
		maxID,
		minID,
		maxID,
		minID,
		maxID,
	).Error
}

func CleanupDedupRows(db *gorm.DB, consumer, eventPrefix string) error {
	if db == nil {
		return fmt.Errorf("nil db")
	}

	return db.Exec(
		"DELETE FROM zfeed_mq_consume_dedup WHERE consumer = ? AND event_id LIKE ?",
		consumer,
		eventPrefix+"%",
	).Error
}

func ensureUniqueIndex(db *gorm.DB, tableName, indexName, createDDL string) error {
	var count int64
	if err := db.Raw(
		`SELECT COUNT(1)
FROM information_schema.statistics
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND index_name = ?`,
		tableName,
		indexName,
	).Scan(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	return db.Exec(createDDL).Error
}

func ensureColumn(db *gorm.DB, tableName, columnName, alterDDL string) error {
	var count int64
	if err := db.Raw(
		`SELECT COUNT(1)
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND table_name = ?
  AND column_name = ?`,
		tableName,
		columnName,
	).Scan(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	return db.Exec(alterDDL).Error
}

func loadEnv() {
	loadEnvOnce.Do(func() {
		if envFile := os.Getenv("ENV_FILE"); envFile != "" {
			_ = godotenv.Load(envFile)
			return
		}

		root := repoRoot()
		if root == "" {
			return
		}

		_ = godotenv.Load(filepath.Join(root, ".env.local"))
		_ = godotenv.Load(filepath.Join(root, ".env"))
	})
}

func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func testDSN() string {
	if dsn := os.Getenv("ZF_MYSQL_TEST_DSN"); dsn != "" {
		return dsn
	}

	host := getenvDefault("MYSQL_HOST", defaultMySQLHost)
	port := getenvDefault("MYSQL_APP_PORT", getenvDefault("MYSQL_PORT", defaultMySQLPort))
	user := getenvDefault("MYSQL_USER", defaultMySQLUser)
	pass := getenvDefault("MYSQL_PASSWORD", defaultMySQLPass)
	dbName := getenvDefault("MYSQL_DATABASE", defaultMySQLDB)
	loc := getenvDefault("MYSQL_LOC", defaultMySQLLoc)

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s",
		user,
		pass,
		host,
		port,
		dbName,
		loc,
	)
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

const createMqConsumeDedupTableDDL = `
CREATE TABLE IF NOT EXISTS zfeed_mq_consume_dedup (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  consumer VARCHAR(64) NOT NULL,
  event_id VARCHAR(128) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uniq_consumer_event (consumer, event_id),
  KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`

const createLikeTableDDL = `
CREATE TABLE IF NOT EXISTS zfeed_like (
  id BIGINT NOT NULL AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  content_id BIGINT NOT NULL,
  content_user_id BIGINT NOT NULL DEFAULT 0,
  status TINYINT NOT NULL COMMENT '10=like,20=cancel',
  last_event_ts BIGINT NOT NULL DEFAULT 0,
  is_deleted TINYINT NOT NULL DEFAULT 0,
  created_by BIGINT NOT NULL DEFAULT 0,
  updated_by BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_content (user_id, content_id),
  KEY idx_content (content_id),
  KEY idx_content_user (content_user_id),
  KEY idx_user_status (user_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
