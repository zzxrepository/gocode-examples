package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP struct {
		Address string `mapstructure:"address"`
	} `mapstructure:"http"`
	Kafka struct {
		BootstrapServers []string `mapstructure:"bootstrap_servers"`
		Topic            string   `mapstructure:"topic"`
		ClientID         string   `mapstructure:"client_id"`
		Producer         struct {
			RequiredAcks string `mapstructure:"required_acks"`
			RetryMax     int    `mapstructure:"retry_max"`
			Idempotent   bool   `mapstructure:"idempotent"`
			Compression  string `mapstructure:"compression"`
		} `mapstructure:"producer"`
		Consumer struct {
			GroupID       string `mapstructure:"group_id"`
			InitialOffset string `mapstructure:"initial_offset"`
			AutoCommit    bool   `mapstructure:"auto_commit"`
		} `mapstructure:"consumer"`
	} `mapstructure:"kafka"`
}

func Default() Config {
	var cfg Config
	cfg.HTTP.Address = "127.0.0.1:18080"
	cfg.Kafka.BootstrapServers = []string{"127.0.0.1:9092"}
	cfg.Kafka.Topic = "order-events-v1"
	cfg.Kafka.ClientID = "order-demo"
	cfg.Kafka.Producer.RequiredAcks = "all"
	cfg.Kafka.Producer.RetryMax = 3
	cfg.Kafka.Producer.Idempotent = true
	cfg.Kafka.Producer.Compression = "none"
	cfg.Kafka.Consumer.GroupID = "order-progress"
	cfg.Kafka.Consumer.InitialOffset = "oldest"
	return cfg
}

func Load(path string) (Config, error) {
	cfg := Default()
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetEnvPrefix("ORDER_DEMO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// 注册键和默认值，使环境变量在 UnmarshalExact 时也参与解码。
	defaults := map[string]any{
		"http.address":                  cfg.HTTP.Address,
		"kafka.bootstrap_servers":       cfg.Kafka.BootstrapServers,
		"kafka.topic":                   cfg.Kafka.Topic,
		"kafka.client_id":               cfg.Kafka.ClientID,
		"kafka.producer.required_acks":  cfg.Kafka.Producer.RequiredAcks,
		"kafka.producer.retry_max":      cfg.Kafka.Producer.RetryMax,
		"kafka.producer.idempotent":     cfg.Kafka.Producer.Idempotent,
		"kafka.producer.compression":    cfg.Kafka.Producer.Compression,
		"kafka.consumer.group_id":       cfg.Kafka.Consumer.GroupID,
		"kafka.consumer.initial_offset": cfg.Kafka.Consumer.InitialOffset,
		"kafka.consumer.auto_commit":    cfg.Kafka.Consumer.AutoCommit,
	}
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("读取配置: %w", err)
	}
	if err := v.UnmarshalExact(&cfg); err != nil {
		return cfg, fmt.Errorf("解析配置: %w", err)
	}
	if len(cfg.Kafka.BootstrapServers) == 0 || strings.TrimSpace(cfg.Kafka.Topic) == "" || strings.TrimSpace(cfg.Kafka.Consumer.GroupID) == "" || cfg.HTTP.Address == "" {
		return cfg, errors.New("bootstrap_servers、topic、group_id 和 http.address 不能为空")
	}
	for i, address := range cfg.Kafka.BootstrapServers {
		cfg.Kafka.BootstrapServers[i] = strings.TrimSpace(address)
		if cfg.Kafka.BootstrapServers[i] == "" {
			return cfg, errors.New("bootstrap_servers 包含空地址")
		}
	}
	_, err := cfg.Sarama()
	return cfg, err
}

func (c Config) Sarama() (*sarama.Config, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.ClientID = c.Kafka.ClientID
	cfg.Metadata.AllowAutoTopicCreation = false
	cfg.Metadata.Timeout = 5 * time.Second
	cfg.Net.DialTimeout = 3 * time.Second
	cfg.Net.ReadTimeout = 5 * time.Second
	cfg.Net.WriteTimeout = 5 * time.Second
	cfg.Producer.Timeout = 3 * time.Second
	switch c.Kafka.Producer.RequiredAcks {
	case "all":
		cfg.Producer.RequiredAcks = sarama.WaitForAll
	case "1":
		cfg.Producer.RequiredAcks = sarama.WaitForLocal
	case "0":
		cfg.Producer.RequiredAcks = sarama.NoResponse
	default:
		return nil, errors.New("required_acks 只能为 all、1 或 0")
	}
	cfg.Producer.Retry.Max = c.Kafka.Producer.RetryMax
	cfg.Producer.Idempotent = c.Kafka.Producer.Idempotent
	cfg.Net.MaxOpenRequests = 1
	cfg.Producer.Partitioner = sarama.NewHashPartitioner
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	switch c.Kafka.Producer.Compression {
	case "none":
		cfg.Producer.Compression = sarama.CompressionNone
	case "gzip":
		cfg.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		cfg.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		cfg.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		cfg.Producer.Compression = sarama.CompressionZSTD
	default:
		return nil, fmt.Errorf("未知 compression %q", c.Kafka.Producer.Compression)
	}
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.AutoCommit.Enable = c.Kafka.Consumer.AutoCommit
	switch c.Kafka.Consumer.InitialOffset {
	case "oldest":
		cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	case "newest":
		cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		return nil, errors.New("initial_offset 只能为 oldest 或 newest")
	}
	cfg.Consumer.IsolationLevel = sarama.ReadCommitted
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	return cfg, cfg.Validate()
}
