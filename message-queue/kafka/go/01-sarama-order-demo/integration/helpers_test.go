package integration

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/config"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/consumer"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/messaging"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/model"
)

type OrderEvent = model.OrderEvent

var consumeGroup = consumer.RunGroup
var orderHandler = consumer.OrderHandler
var sendTransaction = messaging.SendTransaction

func kafkaConfig() *sarama.Config {
	cfg, err := config.Default().Sarama()
	if err != nil {
		panic(err)
	}
	return cfg
}

func orderMessages(topic string) ([]*sarama.ProducerMessage, error) {
	events, err := model.SampleEvents()
	if err != nil {
		return nil, err
	}
	var messages []*sarama.ProducerMessage
	for _, event := range events {
		value, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		msg := messaging.NewMessage(topic, messaging.Record{Key: event.OrderID, Value: value, EventID: event.EventID})
		msg.Metadata = event // 测试用原始事件构造预期值。
		messages = append(messages, msg)
	}
	return messages, nil
}

// 测试建数时直接发送，并保留 Broker 分配的位置用于断言消费组提交。
func sendSync(ctx context.Context, brokers []string, cfg *sarama.Config, messages []*sarama.ProducerMessage) (err error) {
	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, producer.Close()) }()
	for _, msg := range messages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, _, err := producer.SendMessage(msg); err != nil {
			return err
		}
	}
	return nil
}
