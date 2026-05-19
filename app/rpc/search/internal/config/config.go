package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	InteractionRpcClientConf zrpc.RpcClientConf
	RedisConfig              redis.RedisConf
	MySQL                    MySQLConf
	SearchCacheEnabled       bool
	SearchSnapshotEnabled    bool
	SearchHybridRankEnabled  bool
	SearchBackend            string
}

type MySQLConf struct {
	DataSource string
}
