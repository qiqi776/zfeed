package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"

	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/recommend/track"
	"zfeed/app/rpc/count/counterservice"
	"zfeed/app/rpc/interaction/client/favoriteservice"
	"zfeed/app/rpc/interaction/client/followservice"
	"zfeed/app/rpc/interaction/client/likeservice"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/orm"
	"zfeed/pkg/grpcx"
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

	RecommendTrackProducer   track.Producer
	RecommendDailyAggregator *track.DailyAggregator
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

		RecommendTrackProducer:   newRecommendTrackProducer(c.KqProducerConf),
		RecommendDailyAggregator: track.NewDailyAggregator(db),
	}
}

func newRecommendTrackProducer(c config.KqProducerConf) track.Producer {
	if !c.Enabled() {
		return track.NoopProducer{}
	}

	maxRetries := c.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return track.NewKafkaProducer(kq.NewPusher(c.Brokers, c.Topic), maxRetries)
}
