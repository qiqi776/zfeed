package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestContentConfigLoadsWithEnv(t *testing.T) {
	t.Setenv("CONTENT_RPC_LISTEN_ON", "127.0.0.1:5001")
	t.Setenv("PROM_HOST", "127.0.0.1")
	t.Setenv("CONTENT_PROM_PORT", "9291")
	t.Setenv("ETCD_HOST", "127.0.0.1")
	t.Setenv("ETCD_PORT", "12379")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "16379")
	t.Setenv("MYSQL_HOST", "127.0.0.1")
	t.Setenv("MYSQL_APP_PORT", "33306")
	t.Setenv("MYSQL_USER", "zfeed")
	t.Setenv("MYSQL_PASSWORD", "123456")
	t.Setenv("LOG_PATH", "logs")
	t.Setenv("XXL_JOB_ADMIN_ADDR", "http://127.0.0.1:8081/xxl-job-admin")
	t.Setenv("XXL_EXECUTOR_ADDRESS", "127.0.0.1:5005")
	t.Setenv("XXL_EXECUTOR_REGISTRY_ADDRESS", "127.0.0.1:5005")
	t.Setenv("XXL_EXECUTOR_IP", "127.0.0.1")
	t.Setenv("XXL_EXECUTOR_PORT", "5005")
	t.Setenv("XXL_JOB_ACCESS_TOKEN", "default_token")

	var cfg Config
	if err := conf.Load("../../etc/content.yaml", &cfg, conf.UseEnv()); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.ListenOn != "127.0.0.1:5001" {
		t.Fatalf("unexpected listen address: %q", cfg.ListenOn)
	}
	if len(cfg.Etcd.Hosts) != 1 || cfg.Etcd.Hosts[0] != "127.0.0.1:12379" {
		t.Fatalf("unexpected etcd hosts: %v", cfg.Etcd.Hosts)
	}
	if len(cfg.XxlJob.AdminAddresses) != 1 || cfg.XxlJob.AdminAddresses[0] != "http://127.0.0.1:8081/xxl-job-admin" {
		t.Fatalf("unexpected xxl admin addresses: %v", cfg.XxlJob.AdminAddresses)
	}
}
