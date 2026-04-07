package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	RedisConfig    redis.RedisConf
	MySQL          MySQLConf
	KqConsumerConf kq.KqConf
}

type MySQLConf struct {
	DataSource string
}
