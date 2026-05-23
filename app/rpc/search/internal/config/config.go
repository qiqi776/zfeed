package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	InteractionRpcClientConf   zrpc.RpcClientConf
	RedisConfig                redis.RedisConf
	MySQL                      MySQLConf
	SearchCacheEnabled         bool
	SearchSnapshotEnabled      bool
	SearchHybridRankEnabled    bool
	SearchBackend              string
	SearchEngineTrafficPercent int
	SearchEngineCompareEnabled bool
	SearchEngine               SearchEngineConf
	SearchSnapshotTTLSeconds   int
	SearchSnapshotMaxItems     int
	SearchQueryCacheTTLSeconds int
	SearchDocCacheTTLSeconds   int
	SearchQueryCacheMaxPages   int
}

type MySQLConf struct {
	DataSource string
}

type SearchEngineConf struct {
	Endpoint     string
	ContentIndex string
	UserIndex    string
	Username     string
	Password     string
	TimeoutMs    int
}
