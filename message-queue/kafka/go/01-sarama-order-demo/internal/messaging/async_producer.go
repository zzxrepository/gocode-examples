package messaging

import (
	"context"
	"fmt"
	"log"

	"github.com/IBM/sarama"
)

// AsyncPublisher 演示只接收失败结果的异步发送；HTTP 只能报告已受理。
type AsyncPublisher struct {
	producer sarama.AsyncProducer
	topic    string
	done     chan error
}

func NewAsyncPublisher(brokers []string, topic string, cfg *sarama.Config) (*AsyncPublisher, error) {
	copy := *cfg
	copy.Producer.Return.Successes = false
	copy.Producer.Return.Errors = true
	producer, err := sarama.NewAsyncProducer(brokers, &copy)
	if err != nil {
		return nil, err
	}
	p := &AsyncPublisher{producer: producer, topic: topic, done: make(chan error, 1)}
	go func() {
		var firstErr error
		for failure := range producer.Errors() {
			log.Printf("async failed key=%v err=%v", failure.Msg.Key, failure.Err)
			if firstErr == nil {
				firstErr = failure.Err
			}
		}
		p.done <- firstErr
	}()
	return p, nil
}

func (p *AsyncPublisher) Publish(ctx context.Context, record Record) error {
	msg := NewMessage(p.topic, record)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.producer.Input() <- msg:
		log.Printf("async queued event=%s", record.EventID)
		return nil // 仅进入客户端队列，不代表 Broker 已确认。
	}
}

func (p *AsyncPublisher) Close() error {
	p.producer.AsyncClose()
	if err := <-p.done; err != nil {
		return fmt.Errorf("异步生产期间出现错误: %w", err)
	}
	return nil
}
