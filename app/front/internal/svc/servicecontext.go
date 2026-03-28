// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"zfeed/app/front/internal/config"
	"zfeed/app/front/internal/middleware"
	"zfeed/app/rpc/user/client/userservice"
)

type ServiceContext struct {
	Config                        config.Config
	Redis                         *redis.Redis
	ContentRpc                    zrpc.Client
	InteractionRpc                zrpc.Client
	UserRpc                       userservice.UserService
	CountRpc                      zrpc.Client
	UserLoginStatusAuthMiddleware rest.Middleware
	OptionalLoginMiddleware       rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.MustNewRedis(c.RedisConfig)
	contentRpc := zrpc.MustNewClient(c.ContentRpcClientConf)
	interactionRpc := zrpc.MustNewClient(c.InteractionRpcClientConf)
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf)
	countRpc := zrpc.MustNewClient(c.CountRpcClientConf)

	return &ServiceContext{
		Config:                        c,
		Redis:                         rds,
		ContentRpc:                    contentRpc,
		InteractionRpc:                interactionRpc,
		UserRpc:                       userservice.NewUserService(userRpcClient),
		CountRpc:                      countRpc,
		UserLoginStatusAuthMiddleware: middleware.NewUserLoginStatusAuthMiddleware(rds, c).Handle,
		OptionalLoginMiddleware:       middleware.NewOptionalLoginMiddleware(rds, c).Handle,
	}
}
