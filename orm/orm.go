package orm

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN 		 string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  int
}

type DB struct {
	*gorm.DB
}

type ormLog struct {
	LogLevel logger.LogLevel
}

func (l *ormLog) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l
}

func (l *ormLog) Info(ctx context.Context, msg string, data ...interface{}) {
	logger.Default.LogMode(l.LogLevel).Info(ctx, msg, data...)
}

func (l *ormLog) Warn(ctx context.Context, msg string, data ...interface{}) {
	logger.Default.LogMode(l.LogLevel).Warn(ctx, msg, data...)
}

func (l *ormLog) Error(ctx context.Context, msg string, data ...interface{}) {
	logger.Default.LogMode(l.LogLevel).Error(ctx, msg, data...)
}

func (l *ormLog) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	logger.Default.LogMode(l.LogLevel).Trace(ctx, begin, fc, err)
}
