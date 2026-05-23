package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/search-indexer/internal/indexsvc"
	"zfeed/app/rpc/search/search-indexer/internal/indexsync/consumer"
	"zfeed/app/rpc/search/search-indexer/internal/rebuild"
	"zfeed/pkg/envx"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "search-indexer: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	envx.Load()

	command := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		command = args[0]
		args = args[1:]
	}

	fs := flag.NewFlagSet("search-indexer "+command, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configFile := fs.String("f", "etc/search-indexer.yaml", "the config file")
	entity := fs.String("entity", "all", "target entity: all|content|user")
	batchSize := fs.Int("batch-size", 100, "rebuild batch size")
	startID := fs.Int64("start-id", 0, "exclusive start id for rebuild")
	endID := fs.Int64("end-id", 0, "inclusive end id for rebuild")
	dryRun := fs.Bool("dry-run", false, "scan without writing index")
	contentIndex := fs.String("content-index", "", "content index override")
	userIndex := fs.String("user-index", "", "user index override")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var c indexconfig.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if *contentIndex != "" {
		c.IndexEngine.ContentIndex = *contentIndex
	}
	if *userIndex != "" {
		c.IndexEngine.UserIndex = *userIndex
	}
	ctx := indexsvc.NewServiceContext(c)

	switch command {
	case "serve":
		return serve(c, ctx)
	case "rebuild":
		result, err := rebuild.Run(context.Background(), ctx.Repository, ctx.Indexer, rebuild.Options{
			Entity:    *entity,
			BatchSize: *batchSize,
			StartID:   *startID,
			EndID:     *endID,
			DryRun:    *dryRun,
			Out:       os.Stderr,
		})
		if err != nil {
			return err
		}
		fmt.Printf("rebuild entity=%s indexed=%d last_content_id=%d last_user_id=%d elapsed=%s\n", result.Entity, result.Indexed, result.LastContentID, result.LastUserID, result.Elapsed)
		return nil
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func serve(c indexconfig.Config, ctx *indexsvc.ServiceContext) error {
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	for _, mq := range consumer.Consumers(c, context.Background(), ctx) {
		serviceGroup.Add(mq)
	}

	fmt.Printf("Starting search indexer for topic: %s, group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	serviceGroup.Start()
	return nil
}
