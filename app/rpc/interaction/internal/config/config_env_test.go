package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestInteractionConfigLoadsWithEnv(t *testing.T) {
	t.Setenv("INTERACTION_RPC_LISTEN_ON", "127.0.0.1:5002")
	t.Setenv("PROM_HOST", "127.0.0.1")
	t.Setenv("INTERACTION_PROM_PORT", "9293")
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

	var cfg Config
	if err := conf.Load("../../etc/interaction.yaml", &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenOn != "127.0.0.1:5002" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenOn)
	}
	if len(cfg.KqProducerConf.Brokers) != 1 || cfg.KqProducerConf.Brokers[0] != "127.0.0.1:19092" {
		t.Fatalf("unexpected producer brokers: %v", cfg.KqProducerConf.Brokers)
	}
	if len(cfg.KqConsumerConf.Brokers) != 1 || cfg.KqConsumerConf.Brokers[0] != "127.0.0.1:19092" {
		t.Fatalf("unexpected consumer brokers: %v", cfg.KqConsumerConf.Brokers)
	}
}
