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
	"zfeed/app/rpc/search/search-indexer/internal/verify"
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
	contentAlias := fs.String("content-alias", "", "content alias to switch")
	userAlias := fs.String("user-alias", "", "user alias to switch")
	sampleSize := fs.Int("sample-size", 20, "verification sample size")
	topQueries := fs.String("top-queries", "", "comma separated queries for topN overlap verification")
	topN := fs.Int("top-n", 20, "topN size for overlap verification")
	minOverlap := fs.Float64("min-overlap", 0.7, "minimum topN overlap ratio")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var c indexconfig.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if *contentIndex == "" {
		*contentIndex = c.IndexEngine.ContentIndex
	} else {
		c.IndexEngine.ContentIndex = *contentIndex
	}
	if *userIndex == "" {
		*userIndex = c.IndexEngine.UserIndex
	} else {
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
	case "verify":
		result, err := verify.Run(context.Background(), ctx.Repository, ctx.Indexer, verifyOptions(*entity, *contentIndex, *userIndex, *sampleSize, *topQueries, *topN, *minOverlap))
		if err != nil {
			return err
		}
		fmt.Printf("verify ok=%t entity=%s mysql_content=%d index_content=%d mysql_user=%d index_user=%d min_overlap=%.3f elapsed=%s\n", result.OK(), result.Entity, result.MySQLContentCount, result.IndexContentCount, result.MySQLUserCount, result.IndexUserCount, result.MinOverlapObserved, result.Elapsed)
		if !result.OK() {
			return fmt.Errorf("verification failed")
		}
		return nil
	case "switch-alias":
		result, err := verify.SwitchAliasAfterVerify(context.Background(), ctx.Repository, ctx.Indexer, verifyOptions(*entity, *contentIndex, *userIndex, *sampleSize, *topQueries, *topN, *minOverlap), *contentAlias, *userAlias)
		if err != nil {
			return err
		}
		fmt.Printf("switch-alias ok=%t entity=%s content_alias=%s user_alias=%s elapsed=%s\n", result.OK(), result.Entity, *contentAlias, *userAlias, result.Elapsed)
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

func verifyOptions(entity string, contentIndex string, userIndex string, sampleSize int, topQueries string, topN int, minOverlap float64) verify.Options {
	return verify.Options{
		Entity:       entity,
		ContentIndex: contentIndex,
		UserIndex:    userIndex,
		SampleSize:   sampleSize,
		TopQueries:   splitCSV(topQueries),
		TopN:         topN,
		MinOverlap:   minOverlap,
	}
}

func splitCSV(raw string) []string {
	fields := strings.Split(raw, ",")
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(field)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
