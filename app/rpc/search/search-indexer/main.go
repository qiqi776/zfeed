package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/search-indexer/internal/indexsync/consumer"
	"zfeed/app/rpc/search/search-indexer/internal/indexsvc"
	"zfeed/pkg/envx"
)

var configFile = flag.String("f", "etc/search-indexer.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()

	var c indexconfig.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := indexsvc.NewServiceContext(c)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	for _, mq := range consumer.Consumers(c, context.Background(), ctx) {
		serviceGroup.Add(mq)
	}

	fmt.Printf("Starting search indexer for topic: %s, group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	serviceGroup.Start()
}
