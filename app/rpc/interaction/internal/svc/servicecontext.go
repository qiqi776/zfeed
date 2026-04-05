package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/config"
	"zfeed/app/rpc/interaction/internal/mq/producer"
	"zfeed/app/rpc/user/client/userservice"
)

type ServiceContext struct {
	Config       config.Config
	Redis        *redis.Redis
	KqProducer   *kq.Pusher
	LikeProducer producer.EventProducer
	MysqlDb      *gorm.DB
	UserRpc      userservice.UserService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db, err := gorm.Open(mysql.Open(c.MySQL.DataSource), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	kqPusher := kq.NewPusher(c.KqProducerConf.Brokers, c.KqProducerConf.Topic)
	maxRetries := c.KqProducerConf.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	userRpcClient := zrpc.MustNewClient(c.UserRpcClientConf)

	return &ServiceContext{
		Config:       c,
		Redis:        redis.MustNewRedis(c.RedisConfig),
		KqProducer:   kqPusher,
		LikeProducer: producer.NewLikeProducer(kqPusher, maxRetries),
		MysqlDb:      db,
		UserRpc:      userservice.NewUserService(userRpcClient),
	}
}
