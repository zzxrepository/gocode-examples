package messaging

import (
	"context"
	"fmt"
	"log"

	"github.com/IBM/sarama"
)

type SyncPublisher struct {
	producer sarama.SyncProducer
	topic    string
}

func NewSyncPublisher(brokers []string, topic string, cfg *sarama.Config) (*SyncPublisher, error) {
	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}
	return &SyncPublisher{producer: producer, topic: topic}, nil
}

func (p *SyncPublisher) Publish(ctx context.Context, record Record) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	partition, offset, err := p.producer.SendMessage(NewMessage(p.topic, record))
	if err != nil {
		return fmt.Errorf("发布事件: %w", err)
	}
	log.Printf("sync ok event=%s key=%s partition=%d offset=%d", record.EventID, record.Key, partition, offset)
	return nil
}

func (p *SyncPublisher) Close() error { return p.producer.Close() }
