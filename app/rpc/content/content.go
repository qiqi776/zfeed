package main

import (
	"context"
	"flag"
	"fmt"

	"zfeed/app/rpc/content/content"
	"zfeed/app/rpc/content/internal/config"
	"zfeed/app/rpc/content/internal/cron"
	"zfeed/app/rpc/content/internal/server"
	"zfeed/app/rpc/content/internal/svc"
	"zfeed/pkg/envx"
	"zfeed/pkg/grpcx"
	"zfeed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/content.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		content.RegisterContentServiceServer(grpcServer, server.NewContentServiceServer(ctx))
		content.RegisterFeedServiceServer(grpcServer, server.NewFeedServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	grpcx.InstallServerInterceptors(s)
	defer s.Stop()

	xxlCtx, cancelXxl := context.WithCancel(context.Background())
	defer cancelXxl()
	executor := xxljob.NewExecutor(xxljob.Config{
		AppName:          c.XxlJob.AppName,
		Address:          c.XxlJob.Address,
		RegistryAddr:     c.XxlJob.RegistryAddress,
		IP:               c.XxlJob.IP,
		Port:             c.XxlJob.Port,
		AccessToken:      c.XxlJob.AccessToken,
		AdminAddresses:   c.XxlJob.AdminAddresses,
		RegistryInterval: c.XxlJob.RegistryInterval,
		HTTPTimeout:      c.XxlJob.HTTPTimeout,
	})
	cron.Register(xxlCtx, executor, ctx)
	threading.GoSafe(func() {
		if err := executor.Start(xxlCtx); err != nil {
			logx.Errorf("xxl-job executor start failed: %v", err)
		}
	})

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
