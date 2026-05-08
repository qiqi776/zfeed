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
	Config       config.Config
	Redis        *redis.Redis
	KqProducer   *kq.Pusher
	LikeProducer producer.EventProducer
	LikeRelay    service.Service
	MysqlDb      *gorm.DB
	UserRpc      userservice.UserService
	ContentRpc   contentservice.ContentService
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
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf, grpcx.ClientInterceptorOption())
	contentRpcClient := zrpc.MustNewClient(c.ContentRpcClientConf, grpcx.ClientInterceptorOption())
	likeProducer := producer.NewLikeProducer(kqPusher, db, maxRetries)

	return &ServiceContext{
		Config:       c,
		Redis:        redis.MustNewRedis(c.RedisConfig),
		KqProducer:   kqPusher,
		LikeProducer: likeProducer,
		LikeRelay:    producer.NewLikeOutboxRelay(likeProducer),
		MysqlDb:      db,
		UserRpc:      userservice.NewUserService(userRpcClient),
		ContentRpc:   contentservice.NewContentService(contentRpcClient),
	}
}
