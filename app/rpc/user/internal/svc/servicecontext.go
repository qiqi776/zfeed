package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"zfeed/app/rpc/user/internal/config"
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
