package main

import (
	"flag"
	"fmt"

	"zfeed/app/rpc/interaction/interaction"
	"zfeed/app/rpc/interaction/internal/config"
	commentserviceServer "zfeed/app/rpc/interaction/internal/server/commentservice"
	favoriteserviceServer "zfeed/app/rpc/interaction/internal/server/favoriteservice"
	followserviceServer "zfeed/app/rpc/interaction/internal/server/followservice"
	likeserviceServer "zfeed/app/rpc/interaction/internal/server/likeservice"
	"zfeed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/interaction.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		interaction.RegisterLikeServiceServer(grpcServer, likeserviceServer.NewLikeServiceServer(ctx))
		interaction.RegisterFavoriteServiceServer(grpcServer, favoriteserviceServer.NewFavoriteServiceServer(ctx))
		interaction.RegisterCommentServiceServer(grpcServer, commentserviceServer.NewCommentServiceServer(ctx))
		interaction.RegisterFollowServiceServer(grpcServer, followserviceServer.NewFollowServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
