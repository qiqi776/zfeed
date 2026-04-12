package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestUserConfigLoadsWithEnv(t *testing.T) {
	t.Setenv("USER_RPC_LISTEN_ON", "127.0.0.1:5003")
	t.Setenv("PROM_HOST", "127.0.0.1")
	t.Setenv("USER_PROM_PORT", "9294")
	t.Setenv("ETCD_HOST", "127.0.0.1")
	t.Setenv("ETCD_PORT", "12379")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "16379")
	t.Setenv("MYSQL_HOST", "127.0.0.1")
	t.Setenv("MYSQL_APP_PORT", "33306")
	t.Setenv("MYSQL_USER", "zfeed")
	t.Setenv("MYSQL_PASSWORD", "123456")
	t.Setenv("LOG_PATH", "logs")

	var cfg Config
	if err := conf.Load("../../etc/user.yaml", &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenOn != "127.0.0.1:5003" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenOn)
	}
	if len(cfg.Etcd.Hosts) != 1 || cfg.Etcd.Hosts[0] != "127.0.0.1:12379" {
		t.Fatalf("unexpected etcd hosts: %v", cfg.Etcd.Hosts)
	}
	if got := cfg.RedisConfig.Host; got != "127.0.0.1:16379" {
		t.Fatalf("unexpected redis host: %q", got)
	}
}
