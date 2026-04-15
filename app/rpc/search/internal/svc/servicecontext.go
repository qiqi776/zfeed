package svc

import (
	"gorm.io/gorm"
	followservice "zfeed/app/rpc/interaction/client/followservice"

	"zfeed/app/rpc/search/internal/config"
	"zfeed/orm"
	"zfeed/pkg/grpcx"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config    config.Config
	MysqlDb   *gorm.DB
	FollowRpc followservice.FollowService
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "search-rpc",
	})
	interactionClient := zrpc.MustNewClient(c.InteractionRpcClientConf, grpcx.ClientInterceptorOption())

	return &ServiceContext{
		Config:    c,
		MysqlDb:   db,
		FollowRpc: followservice.NewFollowService(interactionClient),
	}
}
