package svc

import (
	"time"

	"zfeed/app/rpc/count/internal/config"
	"zfeed/orm"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

const (
	defaultDelayedCacheInvalidateDelayMs = 200
	defaultDelayedCacheInvalidateWorkers = 4
	defaultDelayedCacheInvalidateQueue   = 1024
)

type ServiceContext struct {
	Config                  config.Config
	Redis                   *redis.Redis
	MysqlDb                 *gorm.DB
	DelayedCacheInvalidator *DelayedCacheInvalidator
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "count-rpc",
	})
	redisClient := redis.MustNewRedis(c.RedisConfig)
	delayMs := c.DelayedCacheInvalidator.DelayMs
	if delayMs <= 0 {
		delayMs = defaultDelayedCacheInvalidateDelayMs
	}
	workers := c.DelayedCacheInvalidator.Workers
	if workers <= 0 {
		workers = defaultDelayedCacheInvalidateWorkers
	}
	queueSize := c.DelayedCacheInvalidator.QueueSize
	if queueSize <= 0 {
		queueSize = defaultDelayedCacheInvalidateQueue
	}

	return &ServiceContext{
		Config:                  c,
		Redis:                   redisClient,
		MysqlDb:                 db,
		DelayedCacheInvalidator: NewDelayedCacheInvalidator(redisClient, time.Duration(delayMs)*time.Millisecond, workers, queueSize),
	}
}

func (s *ServiceContext) Close() {
	if s == nil || s.DelayedCacheInvalidator == nil {
		return
	}
	s.DelayedCacheInvalidator.Close()
}
