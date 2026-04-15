package main

import (
	"context"
	"flag"
	"fmt"

	"zfeed/app/rpc/count/count"
	"zfeed/app/rpc/count/internal/config"
	"zfeed/app/rpc/count/internal/mq/consumer"
	"zfeed/app/rpc/count/internal/server"
	"zfeed/app/rpc/count/internal/svc"
	"zfeed/pkg/envx"
	"zfeed/pkg/grpcx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/count.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		count.RegisterCounterServiceServer(grpcServer, server.NewCounterServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	grpcx.InstallServerInterceptors(s)
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	for _, mq := range consumer.Consumers(c, context.Background(), ctx) {
		serviceGroup.Add(mq)
	}
	serviceGroup.Add(s)
	serviceGroup.Start()
}
