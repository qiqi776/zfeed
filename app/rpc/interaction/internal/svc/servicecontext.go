package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/gorm"

	contentservice "zfeed/app/rpc/content/contentservice"
	"zfeed/app/rpc/interaction/internal/config"
	"zfeed/app/rpc/interaction/internal/mq/producer"
	"zfeed/app/rpc/user/client/userservice"
	"zfeed/orm"
)

type ServiceContext struct {
	Config       config.Config
	Redis        *redis.Redis
	KqProducer   *kq.Pusher
	LikeProducer producer.EventProducer
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
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf)
	contentRpcClient := zrpc.MustNewClient(c.ContentRpcClientConf)

	return &ServiceContext{
		Config:       c,
		Redis:        redis.MustNewRedis(c.RedisConfig),
		KqProducer:   kqPusher,
		LikeProducer: producer.NewLikeProducer(kqPusher, maxRetries),
		MysqlDb:      db,
		UserRpc:      userservice.NewUserService(userRpcClient),
		ContentRpc:   contentservice.NewContentService(contentRpcClient),
	}
}
