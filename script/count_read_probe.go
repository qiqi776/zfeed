package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"zfeed/app/rpc/count/count"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const rpcTimeout = 3 * time.Second

func main() {
	if len(os.Args) < 3 {
		exitf("usage: go run ./script/count_read_probe.go <get|batch|profile> <args>")
	}

	addr := os.Getenv("COUNT_RPC_ADDR")
	if addr == "" {
		addr = os.Getenv("COUNT_RPC_LISTEN_ON")
	}
	if addr == "" {
		exitf("COUNT_RPC_ADDR or COUNT_RPC_LISTEN_ON is required")
	}

	connCtx, connCancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer connCancel()

	conn, err := grpc.DialContext(connCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		exitf("dial count-rpc failed: %v", err)
	}
	defer conn.Close()

	client := count.NewCounterServiceClient(conn)
	switch os.Args[1] {
	case "get":
		runGet(client, os.Args[2:])
	case "batch":
		runBatch(client, os.Args[2:])
	case "profile":
		runProfile(client, os.Args[2:])
	default:
		exitf("unknown mode: %s", os.Args[1])
	}
}

func runGet(client count.CounterServiceClient, args []string) {
	if len(args) != 3 {
		exitf("get requires <biz_type> <target_type> <target_id>")
	}

	bizType := mustParseInt32(args[0])
	targetType := mustParseInt32(args[1])
	targetID := mustParseInt64(args[2])

	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp, err := client.GetCount(ctx, &count.GetCountReq{
		BizType:    count.BizType(bizType),
		TargetType: count.TargetType(targetType),
		TargetId:   targetID,
	})
	if err != nil {
		exitf("get count failed: %v", err)
	}
	fmt.Println(resp.GetValue())
}

func runBatch(client count.CounterServiceClient, args []string) {
	if len(args) != 1 {
		exitf("batch requires <biz:target:id,biz:target:id,...>")
	}

	parts := strings.Split(args[0], ",")
	keys := make([]*count.CountKey, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		fields := strings.Split(part, ":")
		if len(fields) != 3 {
			exitf("invalid batch key: %s", part)
		}
		keys = append(keys, &count.CountKey{
			BizType:    count.BizType(mustParseInt32(fields[0])),
			TargetType: count.TargetType(mustParseInt32(fields[1])),
			TargetId:   mustParseInt64(fields[2]),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp, err := client.BatchGetCount(ctx, &count.BatchGetCountReq{Keys: keys})
	if err != nil {
		exitf("batch get count failed: %v", err)
	}

	for _, item := range resp.GetItems() {
		key := item.GetKey()
		fmt.Printf("%d:%d:%d=%d\n", key.GetBizType(), key.GetTargetType(), key.GetTargetId(), item.GetValue())
	}
}

func runProfile(client count.CounterServiceClient, args []string) {
	if len(args) != 1 {
		exitf("profile requires <user_id>")
	}

	userID := mustParseInt64(args[0])
	ctx, cancel := context.WithTimeout(context.Background(), rpcTimeout)
	defer cancel()

	resp, err := client.GetUserProfileCounts(ctx, &count.GetUserProfileCountsReq{UserId: userID})
	if err != nil {
		exitf("get user profile counts failed: %v", err)
	}

	fmt.Printf("following=%d\n", resp.GetFollowingCount())
	fmt.Printf("followed=%d\n", resp.GetFollowedCount())
	fmt.Printf("like=%d\n", resp.GetLikeCount())
	fmt.Printf("favorite=%d\n", resp.GetFavoriteCount())
}

func mustParseInt32(raw string) int32 {
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		exitf("parse int32 %q failed: %v", raw, err)
	}
	return int32(value)
}

func mustParseInt64(raw string) int64 {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		exitf("parse int64 %q failed: %v", raw, err)
	}
	return value
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
