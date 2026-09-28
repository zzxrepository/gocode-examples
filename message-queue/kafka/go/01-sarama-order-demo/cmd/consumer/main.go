package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/config"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/consumer"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() error {
	path := flag.String("config", "configs/local.yaml", "配置文件路径")
	group := flag.String("group", "", "覆盖配置中的消费组")
	clientID := flag.String("client-id", "", "覆盖客户端标识")
	duration := flag.Duration("duration", 0, "运行时长，0 表示直到 Ctrl+C")
	delay := flag.Duration("delay", 0, "模拟单条业务处理耗时")
	failStatus := flag.String("fail-status", "", "模拟该状态处理失败，例如 paid")
	uncommitted := flag.Bool("read-uncommitted", false, "事务实验：也读取已中止事务")
	flag.Parse()
	if *duration < 0 || *delay < 0 {
		return errors.New("duration/delay 不能为负数")
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	if *group != "" {
		cfg.Kafka.Consumer.GroupID = *group
	}
	if *clientID != "" {
		cfg.Kafka.ClientID = *clientID
	}
	kafkaCfg, err := cfg.Sarama()
	if err != nil {
		return err
	}
	if *uncommitted {
		kafkaCfg.Consumer.IsolationLevel = sarama.ReadUncommitted
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}
	return consumer.RunGroup(ctx, cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, cfg.Kafka.Consumer.GroupID, kafkaCfg,
		consumer.OrderHandler(cfg.Kafka.Consumer.GroupID, cfg.Kafka.ClientID, *delay, *failStatus))
}
