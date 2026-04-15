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
	CountRpcClientConf       zrpc.RpcClientConf
	Oss                      OssConfig
	RedisConfig              redis.RedisConf
	MySQL                    MySQLConf
	XxlJob                   XxlJobConfig
}

type OssConfig struct {
	Provider        string
	Region          string
	BucketName      string
	AccessKeyId     string
	AccessKeySecret string
	Endpoint        string
	UploadDir       string
	PublicHost      string
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
