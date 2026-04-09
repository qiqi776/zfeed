package svc

import (
	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/interaction/client/followservice"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config    config.Config
	Redis     *redis.Redis
	MysqlDb   *gorm.DB
	FollowRpc followservice.FollowService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	followRpc := followservice.NewFollowService(zrpc.MustNewClient(c.InteractionRpcClientConf))

	return &ServiceContext{
		Config:    c,
		Redis:     redis.MustNewRedis(c.RedisConfig),
		MysqlDb:   db,
		FollowRpc: followRpc,
	}
}
