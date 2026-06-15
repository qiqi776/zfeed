package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"

	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/interaction/internal/config"
	"zfeed/app/rpc/interaction/internal/mq/producer"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/orm"
	"zfeed/pkg/grpcx"
)

type ServiceContext struct {
	Config             config.Config
	Redis              *redis.Redis
	KqProducer         *kq.Pusher
	UserActionPusher   *kq.Pusher
	LikeProducer       producer.EventProducer
	UserActionProducer producer.UserActionEventProducer
	LikeRelay          service.Service
	UserActionRelay    service.Service
	MysqlDb            *gorm.DB
	UserRpc            userservice.UserService
	ContentRpc         contentservice.ContentService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "interaction-rpc",
	})

	kqPusher := kq.NewPusher(c.KqProducerConf.Brokers, c.KqProducerConf.Topic)
	maxRetries := c.KqProducerConf.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	userActionPusher := kq.NewPusher(c.KqUserActionProducerConf.Brokers, c.KqUserActionProducerConf.Topic)
	userActionMaxRetries := c.KqUserActionProducerConf.MaxRetries
	if userActionMaxRetries <= 0 {
		userActionMaxRetries = 3
	}
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf, grpcx.ClientInterceptorOption())
	contentRpcClient := zrpc.MustNewClient(c.ContentRpcClientConf, grpcx.ClientInterceptorOption())
	likeProducer := producer.NewLikeProducer(kqPusher, db, maxRetries)
	userActionProducer := producer.NewUserActionProducer(userActionPusher, db, userActionMaxRetries)

	return &ServiceContext{
		Config:             c,
		Redis:              redis.MustNewRedis(c.RedisConfig),
		KqProducer:         kqPusher,
		UserActionPusher:   userActionPusher,
		LikeProducer:       likeProducer,
		UserActionProducer: userActionProducer,
		LikeRelay:          producer.NewLikeOutboxRelay(likeProducer),
		UserActionRelay:    producer.NewUserActionOutboxRelay(userActionProducer),
		MysqlDb:            db,
		UserRpc:            userservice.NewUserService(userRpcClient),
		ContentRpc:         contentservice.NewContentService(contentRpcClient),
	}
}
