package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"

	"zfeed/app/admin-api/internal/config"
	"zfeed/app/admin-api/internal/middleware"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/interaction/client/commentservice"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/grpcx"
)

type ServiceContext struct {
	Config              config.Config
	Redis               *redis.Redis
	ContentRpc          contentservice.ContentService
	CommentRpc          commentservice.CommentService
	UserRpc             userservice.UserService
	CountRpc            counterservice.CounterService
	AdminAuthMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.MustNewRedis(c.RedisConfig)
	contentRpcClient := zrpc.MustNewClient(c.ContentRpcClientConf, grpcx.ClientInterceptorOption())
	contentRpc := contentservice.NewContentService(contentRpcClient)
	interactionRpcClient := zrpc.MustNewClient(c.InteractionRpcClientConf, grpcx.ClientInterceptorOption())
	commentRpc := commentservice.NewCommentService(interactionRpcClient)
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf, grpcx.ClientInterceptorOption())
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(c.CountRpcClientConf, grpcx.ClientInterceptorOption()))

	return &ServiceContext{
		Config:              c,
		Redis:               rds,
		ContentRpc:          contentRpc,
		CommentRpc:          commentRpc,
		UserRpc:             userservice.NewUserService(userRpcClient),
		CountRpc:            countRpc,
		AdminAuthMiddleware: middleware.NewAdminAuthMiddleware(rds, c).Handle,
	}
}
