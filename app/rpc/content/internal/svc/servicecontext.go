package svc

import (
	"zfeed/app/rpc/content/internal/config"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config  config.Config
	Redis   *redis.Redis
	MysqlDb *gorm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:  c,
		Redis:   redis.MustNewRedis(c.RedisConfig),
		MysqlDb: db,
	}
}
