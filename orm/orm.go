package orm

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultMaxIdleConns  = 10
	defaultMaxOpenConns  = 100
	defaultMaxLifetime   = 3600
	defaultSlowThreshold = 200 * time.Millisecond
)

type Config struct {
	DSN           string
	Service       string
	MaxOpenConns  int
	MaxIdleConns  int
	MaxLifetime   int
	SlowThreshold time.Duration
}

type observerLogger struct {
	level logger.LogLevel
}

func (l *observerLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &observerLogger{level: level}
}

func (l *observerLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.level < logger.Info {
		return
	}
	logx.WithContext(ctx).Infof(msg, args...)
}

func (l *observerLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.level < logger.Warn {
		return
	}
	logx.WithContext(ctx).Infof(msg, args...)
}

func (l *observerLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.level < logger.Error {
		return
	}
	logx.WithContext(ctx).Errorf(msg, args...)
}

// Trace is intentionally a no-op. SQL logging and metrics are emitted by the
// observer plugin so table/method metadata can stay structured.
func (l *observerLogger) Trace(context.Context, time.Time, func() (string, int64), error) {}

func NewMysql(conf *Config, plugins ...gorm.Plugin) (*gorm.DB, error) {
	if conf == nil {
		return nil, errors.New("orm config is nil")
	}
	if strings.TrimSpace(conf.DSN) == "" {
		return nil, errors.New("orm dsn is empty")
	}

	if conf.MaxIdleConns <= 0 {
		conf.MaxIdleConns = defaultMaxIdleConns
	}
	if conf.MaxOpenConns <= 0 {
		conf.MaxOpenConns = defaultMaxOpenConns
	}
	if conf.MaxLifetime <= 0 {
		conf.MaxLifetime = defaultMaxLifetime
	}
	if conf.SlowThreshold <= 0 {
		conf.SlowThreshold = defaultSlowThreshold
	}

	db, err := gorm.Open(mysql.Open(conf.DSN), &gorm.Config{
		Logger: &observerLogger{level: logger.Info},
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(conf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(conf.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Second * time.Duration(conf.MaxLifetime))

	for _, plugin := range plugins {
		if plugin == nil {
			continue
		}
		if err := db.Use(plugin); err != nil {
			return nil, err
		}
	}

	if err := db.Use(NewObserverPlugin(conf.Service, conf.SlowThreshold)); err != nil {
		return nil, err
	}

	return db, nil
}

func MustNewMysql(conf *Config, plugins ...gorm.Plugin) *gorm.DB {
	db, err := NewMysql(conf, plugins...)
	if err != nil {
		logx.Errorf("init mysql failed: %v", err)
		panic(err)
	}

	return db
}
