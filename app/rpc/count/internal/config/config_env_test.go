package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestCountConfigLoadsWithEnv(t *testing.T) {
	t.Setenv("COUNT_RPC_LISTEN_ON", "127.0.0.1:5004")
	t.Setenv("PROM_HOST", "127.0.0.1")
	t.Setenv("COUNT_PROM_PORT", "9292")
	t.Setenv("ETCD_HOST", "127.0.0.1")
	t.Setenv("ETCD_PORT", "12379")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "16379")
	t.Setenv("MYSQL_HOST", "127.0.0.1")
	t.Setenv("MYSQL_APP_PORT", "33306")
	t.Setenv("MYSQL_USER", "zfeed")
	t.Setenv("MYSQL_PASSWORD", "123456")
	t.Setenv("KAFKA_BROKERS", "127.0.0.1:19092")
	t.Setenv("LOG_PATH", "logs")
	t.Setenv("OTEL_ENDPOINT", "127.0.0.1:4317")

	var cfg Config
	if err := conf.Load("../../etc/count.yaml", &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenOn != "127.0.0.1:5004" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenOn)
	}
	if len(cfg.Etcd.Hosts) != 1 || cfg.Etcd.Hosts[0] != "127.0.0.1:12379" {
		t.Fatalf("unexpected etcd hosts: %v", cfg.Etcd.Hosts)
	}
	if len(cfg.KqConsumerConf.Brokers) != 1 || cfg.KqConsumerConf.Brokers[0] != "127.0.0.1:19092" {
		t.Fatalf("unexpected consumer brokers: %v", cfg.KqConsumerConf.Brokers)
	}
	if cfg.Telemetry.Name != "count-rpc" || cfg.Telemetry.Endpoint != "127.0.0.1:4317" {
		t.Fatalf("unexpected telemetry config: %+v", cfg.Telemetry)
	}
}
