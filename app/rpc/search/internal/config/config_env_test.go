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
	if cfg.Telemetry.Name != "search-rpc" || cfg.Telemetry.Endpoint != "127.0.0.1:4317" || !cfg.Telemetry.Disabled {
		t.Fatalf("unexpected telemetry config: %+v", cfg.Telemetry)
	}
}
