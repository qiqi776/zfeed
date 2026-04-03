package main

import (
	"context"
	"flag"
	"fmt"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/config"
	"zfeed/app/rpc/interaction/internal/mq/consumer"
	commentserviceServer "zfeed/app/rpc/interaction/internal/server/commentservice"
	favoriteserviceServer "zfeed/app/rpc/interaction/internal/server/favoriteservice"
	followserviceServer "zfeed/app/rpc/interaction/internal/server/followservice"
	likeserviceServer "zfeed/app/rpc/interaction/internal/server/likeservice"
	"zfeed/app/rpc/interaction/internal/svc"
	"zfeed/pkg/envx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/interaction.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	rpcServer := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		interaction.RegisterLikeServiceServer(grpcServer, likeserviceServer.NewLikeServiceServer(ctx))
		interaction.RegisterFavoriteServiceServer(grpcServer, favoriteserviceServer.NewFavoriteServiceServer(ctx))
		interaction.RegisterCommentServiceServer(grpcServer, commentserviceServer.NewCommentServiceServer(ctx))
		interaction.RegisterFollowServiceServer(grpcServer, followserviceServer.NewFollowServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer rpcServer.Stop()

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	for _, mq := range consumer.Consumers(c, context.Background(), ctx) {
		serviceGroup.Add(mq)
	}
	serviceGroup.Add(rpcServer)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	fmt.Printf("Starting mq consumer for topic: %s, group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	serviceGroup.Start()
}
