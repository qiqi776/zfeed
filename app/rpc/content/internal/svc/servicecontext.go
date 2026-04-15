package svc

import (
	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/interaction/client/likeservice"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/grpcx"
	"zfeed/orm"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"
)

type ServiceContext struct {
	Config      config.Config
	Redis       *redis.Redis
	MysqlDb     *gorm.DB
	FollowRpc   followservice.FollowService
	FavoriteRpc favoriteservice.FavoriteService
	LikeRpc     likeservice.LikeService
	UserRpc     userservice.UserService
	CountRpc    counterservice.CounterService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "content-rpc",
	})

	interactionRpcClient := zrpc.MustNewClient(c.InteractionRpcClientConf, grpcx.ClientInterceptorOption())
	followRpc := followservice.NewFollowService(interactionRpcClient)
	favoriteRpc := favoriteservice.NewFavoriteService(interactionRpcClient)
	likeRpc := likeservice.NewLikeService(interactionRpcClient)
	userRpc := userservice.NewUserService(zrpc.MustNewClient(c.UserRpcClientConf, grpcx.ClientInterceptorOption()))
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(c.CountRpcClientConf, grpcx.ClientInterceptorOption()))

	return &ServiceContext{
		Config:      c,
		Redis:       redis.MustNewRedis(c.RedisConfig),
		MysqlDb:     db,
		FollowRpc:   followRpc,
		FavoriteRpc: favoriteRpc,
		LikeRpc:     likeRpc,
		UserRpc:     userRpc,
		CountRpc:    countRpc,
	}
}
