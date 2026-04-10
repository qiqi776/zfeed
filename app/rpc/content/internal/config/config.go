package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	InteractionRpcClientConf zrpc.RpcClientConf
	UserRpcClientConf        zrpc.RpcClientConf
	RedisConfig              redis.RedisConf
	MySQL                    MySQLConf
	XxlJob                   XxlJobConfig
}

type MySQLConf struct {
	DataSource string
}

type XxlJobConfig struct {
	AppName          string
	Address          string
	RegistryAddress  string
	IP               string
	Port             int
	AccessToken      string
	AdminAddresses   []string
	RegistryInterval time.Duration
	HTTPTimeout      time.Duration
}
