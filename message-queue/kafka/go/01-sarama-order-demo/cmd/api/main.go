package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/config"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/controller"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/messaging"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/router"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run() (err error) {
	path := flag.String("config", "configs/local.yaml", "YAML 配置路径")
	flag.Parse()
	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	kafkaCfg, err := cfg.Sarama()
	if err != nil {
		return err
	}
	// HTTP 成功响应需要 Broker 确认；禁止用 acks=0 运行订单 API。
	if cfg.Kafka.Producer.RequiredAcks != "all" {
		return errors.New("订单 API 要求 required_acks=all")
	}
	publisher, err := messaging.NewSyncPublisher(cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, kafkaCfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, publisher.Close()) }()
	asyncPublisher, err := messaging.NewAsyncPublisher(cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, kafkaCfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, asyncPublisher.Close()) }()
	callbackPublisher, err := messaging.NewCallbackPublisher(cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, kafkaCfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, callbackPublisher.Close()) }()
	orders := service.NewOrderService(publisher)
	handler := router.New(
		controller.NewOrderController(orders),
		controller.NewAsyncOrderController(service.NewOrderService(asyncPublisher)),
		controller.NewAsyncOrderController(service.NewOrderService(callbackPublisher)),
		controller.NewTransactionController(cfg.Kafka.BootstrapServers, cfg.Kafka.Topic, kafkaCfg),
	)
	server := &http.Server{
		Addr: cfg.HTTP.Address, Handler: handler,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	listener, err := net.Listen("tcp", cfg.HTTP.Address)
	if err != nil {
		return err
	}
	go func() { done <- server.Serve(listener) }()
	log.Printf("order API listening on %s, topic=%s bootstrap_servers=%v", cfg.HTTP.Address, cfg.Kafka.Topic, cfg.Kafka.BootstrapServers)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return errors.Join(err, server.Close())
	}
	if err := <-done; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
