package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	ContentRpcClientConf     zrpc.RpcClientConf
	InteractionRpcClientConf zrpc.RpcClientConf
	UserRpcClientConf        zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
	SessionTTL               int64
	RedisConfig              redis.RedisConf
	MySQL                    MySQLConf
}

type MySQLConf struct {
	DataSource string
}
