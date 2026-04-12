package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"

	"zfeed/app/rpc/user/internal/config"
	"zfeed/orm"
)

type ServiceContext struct {
	Config  config.Config
	Redis   *redis.Redis
	MysqlDb *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "user-rpc",
	})

	return &ServiceContext{
		Config:  c,
		Redis:   redis.MustNewRedis(c.RedisConfig),
		MysqlDb: db,
	}
}
