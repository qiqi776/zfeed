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
}
