package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestSearchConfigLoadsWithEnv(t *testing.T) {
	t.Setenv("SEARCH_RPC_LISTEN_ON", "127.0.0.1:5006")
	t.Setenv("PROM_HOST", "127.0.0.1")
	t.Setenv("SEARCH_PROM_PORT", "9295")
	t.Setenv("ETCD_HOST", "127.0.0.1")
	t.Setenv("ETCD_PORT", "12379")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "16379")
	t.Setenv("SEARCH_CACHE_ENABLED", "true")
	t.Setenv("SEARCH_SNAPSHOT_ENABLED", "false")
	t.Setenv("SEARCH_HYBRID_RANK_ENABLED", "true")
	t.Setenv("SEARCH_BACKEND", "mysql")
	t.Setenv("SEARCH_ENGINE_TRAFFIC_PERCENT", "10")
	t.Setenv("SEARCH_INDEX_COMPARE_ENABLED", "true")
	t.Setenv("SEARCH_INDEX_ENGINE_ENDPOINT", "http://127.0.0.1:19200")
	t.Setenv("SEARCH_INDEX_CONTENT_INDEX", "contents")
	t.Setenv("SEARCH_INDEX_USER_INDEX", "users")
	t.Setenv("SEARCH_INDEX_ENGINE_USERNAME", "elastic")
	t.Setenv("SEARCH_INDEX_ENGINE_PASSWORD", "secret")
	t.Setenv("SEARCH_INDEX_ENGINE_TIMEOUT_MS", "2000")
	t.Setenv("SEARCH_SNAPSHOT_TTL_SECONDS", "60")
	t.Setenv("SEARCH_SNAPSHOT_MAX_ITEMS", "100")
	t.Setenv("SEARCH_QUERY_CACHE_TTL_SECONDS", "60")
	t.Setenv("SEARCH_DOC_CACHE_TTL_SECONDS", "600")
	t.Setenv("SEARCH_QUERY_CACHE_MAX_PAGES", "3")
	t.Setenv("MYSQL_HOST", "127.0.0.1")
	t.Setenv("MYSQL_APP_PORT", "33306")
	t.Setenv("MYSQL_USER", "zfeed")
	t.Setenv("MYSQL_PASSWORD", "123456")
	t.Setenv("LOG_PATH", "logs")
	t.Setenv("OTEL_DISABLED", "true")
	t.Setenv("OTEL_ENDPOINT", "127.0.0.1:4317")

	var cfg Config
	if err := conf.Load("../../etc/search.yaml", &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenOn != "127.0.0.1:5006" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenOn)
	}
	if len(cfg.Etcd.Hosts) != 1 || cfg.Etcd.Hosts[0] != "127.0.0.1:12379" {
		t.Fatalf("unexpected etcd hosts: %v", cfg.Etcd.Hosts)
	}
	if len(cfg.InteractionRpcClientConf.Etcd.Hosts) != 1 || cfg.InteractionRpcClientConf.Etcd.Hosts[0] != "127.0.0.1:12379" {
		t.Fatalf("unexpected interaction etcd hosts: %v", cfg.InteractionRpcClientConf.Etcd.Hosts)
	}
	if got := cfg.RedisConfig.Host; got != "127.0.0.1:16379" {
		t.Fatalf("unexpected redis host: %q", got)
	}
	if !cfg.SearchCacheEnabled || cfg.SearchSnapshotEnabled || !cfg.SearchHybridRankEnabled || cfg.SearchBackend != "mysql" {
		t.Fatalf("unexpected search feature config: %+v", cfg)
	}
	if cfg.SearchEngineTrafficPercent != 10 {
		t.Fatalf("unexpected search engine traffic percent: %+v", cfg)
	}
	if !cfg.SearchEngineCompareEnabled ||
		cfg.SearchEngine.Endpoint != "http://127.0.0.1:19200" ||
		cfg.SearchEngine.ContentIndex != "contents" ||
		cfg.SearchEngine.UserIndex != "users" ||
		cfg.SearchEngine.Username != "elastic" ||
		cfg.SearchEngine.Password != "secret" ||
		cfg.SearchEngine.TimeoutMs != 2000 {
		t.Fatalf("unexpected search engine config: %+v", cfg.SearchEngine)
	}
	if cfg.SearchSnapshotTTLSeconds != 60 || cfg.SearchSnapshotMaxItems != 100 {
		t.Fatalf("unexpected search snapshot config: %+v", cfg)
	}
	if cfg.SearchQueryCacheTTLSeconds != 60 || cfg.SearchDocCacheTTLSeconds != 600 || cfg.SearchQueryCacheMaxPages != 3 {
		t.Fatalf("unexpected search cache config: %+v", cfg)
	}
	if cfg.Telemetry.Name != "search-rpc" || cfg.Telemetry.Endpoint != "127.0.0.1:4317" || !cfg.Telemetry.Disabled {
		t.Fatalf("unexpected telemetry config: %+v", cfg.Telemetry)
	}
}
