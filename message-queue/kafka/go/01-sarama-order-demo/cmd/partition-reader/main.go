package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() (err error) {
	path := flag.String("config", "configs/local.yaml", "配置文件路径")
	partition := flag.Int("partition", 0, "直接读取的分区")
	offset := flag.Int64("offset", sarama.OffsetOldest, "开始位置：-2 最早，-1 最新，或具体 offset")
	flag.Parse()
	if *partition < 0 || int64(*partition) > 2147483647 || *offset < -2 {
		return errors.New("partition/offset 不合法")
	}
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	kafkaCfg, err := cfg.Sarama()
	if err != nil {
		return err
	}
	reader, err := sarama.NewConsumer(cfg.Kafka.BootstrapServers, kafkaCfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	pc, err := reader.ConsumePartition(cfg.Kafka.Topic, int32(*partition), *offset)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, pc.Close()) }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case failure, ok := <-pc.Errors():
			if !ok {
				return nil
			}
			return failure
		case msg, ok := <-pc.Messages():
			if !ok {
				return nil
			}
			fmt.Printf("partition=%d offset=%d key=%s value=%s\n", msg.Partition, msg.Offset, msg.Key, msg.Value)
		}
	}
}
