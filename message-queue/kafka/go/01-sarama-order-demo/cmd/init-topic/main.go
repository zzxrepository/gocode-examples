package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (err error) {
	path := flag.String("config", "configs/local.yaml", "配置文件路径")
	flag.Parse()
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	kafkaCfg, err := cfg.Sarama()
	if err != nil {
		return err
	}
	admin, err := sarama.NewClusterAdmin(cfg.Kafka.BootstrapServers, kafkaCfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, admin.Close()) }()
	err = admin.CreateTopic(cfg.Kafka.Topic, &sarama.TopicDetail{NumPartitions: 3, ReplicationFactor: 1}, false)
	if errors.Is(err, sarama.ErrTopicAlreadyExists) {
		fmt.Println("Topic 已存在，保留分区和数据:", cfg.Kafka.Topic)
		return nil
	}
	if err == nil {
		fmt.Println("created topic=" + cfg.Kafka.Topic + " partitions=3 replication-factor=1")
	}
	return err
}
