package svc

import (
	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/user/client/userservice"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config      config.Config
	Redis       *redis.Redis
	MysqlDb     *gorm.DB
	FollowRpc   followservice.FollowService
	FavoriteRpc favoriteservice.FavoriteService
	UserRpc     userservice.UserService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	interactionRpcClient := zrpc.MustNewClient(c.InteractionRpcClientConf)
	followRpc := followservice.NewFollowService(interactionRpcClient)
	favoriteRpc := favoriteservice.NewFavoriteService(interactionRpcClient)
	userRpc := userservice.NewUserService(zrpc.MustNewClient(c.UserRpcClientConf))

	return &ServiceContext{
		Config:      c,
		Redis:       redis.MustNewRedis(c.RedisConfig),
		MysqlDb:     db,
		FollowRpc:   followRpc,
		FavoriteRpc: favoriteRpc,
		UserRpc:     userRpc,
	}
}
