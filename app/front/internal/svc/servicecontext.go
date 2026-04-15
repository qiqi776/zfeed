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
	"zfeed/app/rpc/content/feedservice"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/interaction/client/commentservice"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/interaction/client/likeservice"
	"zfeed/app/rpc/search/searchservice"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/pkg/grpcx"
)

type ServiceContext struct {
	Config                        config.Config
	Redis                         *redis.Redis
	ContentRpc                    contentservice.ContentService
	FeedRpc                       feedservice.FeedService
	CommentRpc                    commentservice.CommentService
	FavoriteRpc                   favoriteservice.FavoriteService
	FollowRpc                     followservice.FollowService
	LikeRpc                       likeservice.LikeService
	UserRpc                       userservice.UserService
	CountRpc                      counterservice.CounterService
	SearchRpc                     searchservice.SearchService
	UserLoginStatusAuthMiddleware rest.Middleware
	OptionalLoginMiddleware       rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.MustNewRedis(c.RedisConfig)
	contentRpcClient := zrpc.MustNewClient(c.ContentRpcClientConf, grpcx.ClientInterceptorOption())
	contentRpc := contentservice.NewContentService(contentRpcClient)
	feedRpc := feedservice.NewFeedService(contentRpcClient)
	interactionRpcClient := zrpc.MustNewClient(c.InteractionRpcClientConf, grpcx.ClientInterceptorOption())
	likeRpc := likeservice.NewLikeService(interactionRpcClient)
	commentRpc := commentservice.NewCommentService(interactionRpcClient)
	favoriteRpc := favoriteservice.NewFavoriteService(interactionRpcClient)
	followRpc := followservice.NewFollowService(interactionRpcClient)
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf, grpcx.ClientInterceptorOption())
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(c.CountRpcClientConf, grpcx.ClientInterceptorOption()))
	searchRpc := searchservice.NewSearchService(zrpc.MustNewClient(c.SearchRpcClientConf, grpcx.ClientInterceptorOption()))

	return &ServiceContext{
		Config:                        c,
		Redis:                         rds,
		ContentRpc:                    contentRpc,
		FeedRpc:                       feedRpc,
		CommentRpc:                    commentRpc,
		FavoriteRpc:                   favoriteRpc,
		FollowRpc:                     followRpc,
		LikeRpc:                       likeRpc,
		UserRpc:                       userservice.NewUserService(userRpcClient),
		CountRpc:                      countRpc,
		SearchRpc:                     searchRpc,
		UserLoginStatusAuthMiddleware: middleware.NewUserLoginStatusAuthMiddleware(rds, c).Handle,
		OptionalLoginMiddleware:       middleware.NewOptionalLoginMiddleware(rds, c).Handle,
	}
}
