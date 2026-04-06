package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	UserRpcClientConf    zrpc.RpcClientConf
	ContentRpcClientConf zrpc.RpcClientConf
	RedisConfig          redis.RedisConf
	KqProducerConf       KqProducerConf
	KqConsumerConf       kq.KqConf
	MySQL                MySQLConf
}

type KqProducerConf struct {
	Brokers    []string
	Topic      string
	MaxRetries int
}

type MySQLConf struct {
	DataSource string
}
