package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestViperEnvironmentOverridesFileAndDefaults(t *testing.T) {
	path := writeConfig(t, "http:\n  address: '127.0.0.1:18080'\nkafka:\n  topic: file-topic\n")
	t.Setenv("ORDER_DEMO_HTTP_ADDRESS", "127.0.0.1:18081")
	t.Setenv("ORDER_DEMO_KAFKA_TOPIC", "env-topic")
	t.Setenv("ORDER_DEMO_KAFKA_BOOTSTRAP_SERVERS", "127.0.0.1:19092,127.0.0.1:29092")
	t.Setenv("ORDER_DEMO_KAFKA_CONSUMER_AUTO_COMMIT", "true")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Address != "127.0.0.1:18081" || cfg.Kafka.Topic != "env-topic" {
		t.Fatalf("环境变量未覆盖文件: %+v", cfg)
	}
	if len(cfg.Kafka.BootstrapServers) != 2 || cfg.Kafka.BootstrapServers[1] != "127.0.0.1:29092" {
		t.Fatalf("地址列表解码错误: %v", cfg.Kafka.BootstrapServers)
	}
	if !cfg.Kafka.Consumer.AutoCommit || cfg.Kafka.Producer.RequiredAcks != "all" {
		t.Fatal("环境变量或默认值没有生效")
	}
}

func TestRejectUnknownAndIncompatibleConfig(t *testing.T) {
	for _, content := range []string{
		"kafka:\n  bootstrap_server: [localhost:9092]\n",
		"kafka:\n  producer:\n    required_acks: '0'\n    idempotent: true\n",
		"kafka:\n  bootstrap_servers: ['']\n",
	} {
		if _, err := Load(writeConfig(t, content)); err == nil {
			t.Fatalf("应拒绝配置: %s", content)
		}
	}
}
