// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"

	"zfeed/app/front/internal/config"
	"zfeed/app/front/internal/middleware"
	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/interaction/client/commentservice"
	"zfeed/app/rpc/interaction/client/likeservice"
	"zfeed/app/rpc/user/client/userservice"
)

type ServiceContext struct {
	Config                        config.Config
	Redis                         *redis.Redis
	ContentRpc                    contentservice.ContentService
	CommentRpc                    commentservice.CommentService
	LikeRpc                       likeservice.LikeService
	UserRpc                       userservice.UserService
	CountRpc                      zrpc.Client
	UserLoginStatusAuthMiddleware rest.Middleware
	OptionalLoginMiddleware       rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.MustNewRedis(c.RedisConfig)
	contentRpc := contentservice.NewContentService(zrpc.MustNewClient(c.ContentRpcClientConf))
	interactionRpcClient := zrpc.MustNewClient(c.InteractionRpcClientConf)
	likeRpc := likeservice.NewLikeService(interactionRpcClient)
	commentRpc := commentservice.NewCommentService(interactionRpcClient)
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf)
	countRpc := zrpc.MustNewClient(c.CountRpcClientConf)

	return &ServiceContext{
		Config:                        c,
		Redis:                         rds,
		ContentRpc:                    contentRpc,
		CommentRpc:                    commentRpc,
		LikeRpc:                       likeRpc,
		UserRpc:                       userservice.NewUserService(userRpcClient),
		CountRpc:                      countRpc,
		UserLoginStatusAuthMiddleware: middleware.NewUserLoginStatusAuthMiddleware(rds, c).Handle,
		OptionalLoginMiddleware:       middleware.NewOptionalLoginMiddleware(rds, c).Handle,
	}
}
