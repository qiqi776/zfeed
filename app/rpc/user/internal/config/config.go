package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	InteractionRpcClientConf zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
	RedisConfig              redis.RedisConf
	MySQL                    MySQLConf
	SessionTTL               int64
	Oss                      OssConfig
}

type MySQLConf struct {
	DataSource string
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