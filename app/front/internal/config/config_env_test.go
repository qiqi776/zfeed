package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestFrontConfigLoadsWithEnv(t *testing.T) {
	t.Setenv("FRONT_API_HOST", "127.0.0.1")
	t.Setenv("FRONT_API_PORT", "5000")
	t.Setenv("PROM_HOST", "127.0.0.1")
	t.Setenv("PROM_PORT", "9290")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "16379")
	t.Setenv("ETCD_HOST", "127.0.0.1")
	t.Setenv("ETCD_PORT", "12379")
	t.Setenv("MYSQL_HOST", "127.0.0.1")
	t.Setenv("MYSQL_APP_PORT", "33306")
	t.Setenv("MYSQL_USER", "zfeed")
	t.Setenv("MYSQL_PASSWORD", "123456")
	t.Setenv("LOG_PATH", "logs")
	t.Setenv("OTEL_ENDPOINT", "127.0.0.1:4317")

	var cfg Config
	if err := conf.Load("../../etc/front-api.yaml", &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Host != "127.0.0.1" || cfg.Port != 5000 {
		t.Fatalf("unexpected front config: host=%q port=%d", cfg.Host, cfg.Port)
	}
	if got := cfg.RedisConfig.Host; got != "127.0.0.1:16379" {
		t.Fatalf("unexpected redis host: %q", got)
	}
	if len(cfg.SearchRpcClientConf.Etcd.Hosts) != 1 || cfg.SearchRpcClientConf.Etcd.Hosts[0] != "127.0.0.1:12379" {
		t.Fatalf("unexpected search rpc etcd hosts: %v", cfg.SearchRpcClientConf.Etcd.Hosts)
	}
	if cfg.Telemetry.Name != "front-api" || cfg.Telemetry.Endpoint != "127.0.0.1:4317" {
		t.Fatalf("unexpected telemetry config: %+v", cfg.Telemetry)
	}
}
