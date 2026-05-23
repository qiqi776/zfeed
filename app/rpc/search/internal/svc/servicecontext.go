package svc

import (
	"context"

	"gorm.io/gorm"
	followservice "zfeed/app/rpc/interaction/client/followservice"

	"zfeed/app/rpc/search/internal/backend"
	"zfeed/app/rpc/search/internal/config"
	"zfeed/app/rpc/search/internal/querynorm"
	"zfeed/orm"
	"zfeed/pkg/grpcx"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config          config.Config
	Redis           *redis.Redis
	MysqlDb         *gorm.DB
	FollowRpc       followservice.FollowService
	BackendFactory  backend.Factory
	QueryNormalizer querynorm.Normalizer
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "search-rpc",
	})
	redisClient := redis.MustNewRedis(c.RedisConfig)
	interactionClient := zrpc.MustNewClient(c.InteractionRpcClientConf, grpcx.ClientInterceptorOption())

	return &ServiceContext{
		Config:    c,
		Redis:     redisClient,
		MysqlDb:   db,
		FollowRpc: followservice.NewFollowService(interactionClient),
		BackendFactory: backend.NewFactoryWithEngineConfig(db, c.SearchBackend, backend.EngineConfig{
			Endpoint:       c.SearchEngine.Endpoint,
			ContentIndex:   c.SearchEngine.ContentIndex,
			UserIndex:      c.SearchEngine.UserIndex,
			Username:       c.SearchEngine.Username,
			Password:       c.SearchEngine.Password,
			TimeoutMs:      c.SearchEngine.TimeoutMs,
			CompareEnabled: c.SearchEngineCompareEnabled,
		}, c.SearchEngineTrafficPercent),
		QueryNormalizer: querynorm.NewDefaultNormalizer(),
	}
}

func (s *ServiceContext) SearchBackend(ctx context.Context) backend.SearchBackend {
	if s != nil && s.BackendFactory != nil {
		return s.BackendFactory.Backend(ctx)
	}
	if s == nil {
		return backend.NewMySQLBackend(nil)
	}
	return backend.NewMySQLBackend(s.MysqlDb)
}

func (s *ServiceContext) ConfiguredSearchBackend() string {
	if s != nil && s.BackendFactory != nil {
		return s.BackendFactory.ConfiguredBackend()
	}
	return backend.NameMySQL
}

func (s *ServiceContext) EffectiveSearchBackend() string {
	if s != nil && s.BackendFactory != nil {
		return s.BackendFactory.EffectiveBackend()
	}
	return backend.NameMySQL
}

func (s *ServiceContext) NormalizeQuery(raw string) querynorm.Query {
	if s != nil && s.QueryNormalizer != nil {
		return s.QueryNormalizer.Normalize(raw)
	}
	return querynorm.NewDefaultNormalizer().Normalize(raw)
}
