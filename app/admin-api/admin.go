package main

import (
	"context"
	"flag"
	"fmt"

	"zfeed/app/admin-api/internal/config"
	"zfeed/app/admin-api/internal/handler"
	"zfeed/app/admin-api/internal/svc"
	"zfeed/pkg/envx"
	"zfeed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/admin-api.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	httpx.SetErrorHandlerCtx(func(_ context.Context, err error) (int, any) {
		return errorx.ResponseFromError(err)
	})

	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting admin-api at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
