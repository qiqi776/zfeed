package svc

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"zfeed/app/rpc/interaction/internal/config"
	"zfeed/app/rpc/interaction/internal/mq/producer"
)

type ServiceContext struct {
	Config       config.Config
	Redis        *redis.Redis
	KqProducer   *kq.Pusher
	LikeProducer producer.EventProducer
	MysqlDb      *gorm.DB
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

	return &ServiceContext{
		Config:       c,
		Redis:        redis.MustNewRedis(c.RedisConfig),
		KqProducer:   kqPusher,
		LikeProducer: producer.NewLikeProducer(kqPusher, maxRetries),
		MysqlDb:      db,
	}
}
