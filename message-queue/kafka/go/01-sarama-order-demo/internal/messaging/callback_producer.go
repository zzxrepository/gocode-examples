package messaging

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
)

// CallbackPublisher 独立演示成功与失败结果处理，避免与同步路径混在一个模式分支里。
type CallbackPublisher struct {
	producer  sarama.AsyncProducer
	topic     string
	done      chan error
	mu        sync.RWMutex
	closed    bool
	pending   sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
}

func NewCallbackPublisher(brokers []string, topic string, cfg *sarama.Config) (*CallbackPublisher, error) {
	copy := *cfg
	copy.Producer.Return.Successes = true
	copy.Producer.Return.Errors = true
	producer, err := sarama.NewAsyncProducer(brokers, &copy)
	if err != nil {
		return nil, err
	}
	p := &CallbackPublisher{producer: producer, topic: topic, done: make(chan error, 1)}
	go func() {
		var firstErr error
		successes, failures := producer.Successes(), producer.Errors()
		for successes != nil || failures != nil {
			select {
			case msg, ok := <-successes:
				if !ok {
					successes = nil
					continue
				}
				log.Printf("callback ok event=%v key=%v partition=%d offset=%d", msg.Metadata, msg.Key, msg.Partition, msg.Offset)
				p.pending.Done()
			case failure, ok := <-failures:
				if !ok {
					failures = nil
					continue
				}
				log.Printf("callback failed key=%v err=%v", failure.Msg.Key, failure.Err)
				if firstErr == nil {
					firstErr = failure.Err
				}
				p.pending.Done()
			}
		}
		p.done <- firstErr
	}()
	return p, nil
}

func (p *CallbackPublisher) Publish(ctx context.Context, record Record) error {
	// 与 Close 互斥，禁止关闭开始后继续向 Sarama 输入通道写入。
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return errors.New("生产者已关闭")
	}
	p.pending.Add(1)
	msg := NewMessage(p.topic, record)
	select {
	case <-ctx.Done():
		p.pending.Done()
		return ctx.Err()
	case p.producer.Input() <- msg:
		log.Printf("callback queued event=%s", record.EventID)
		return nil
	}
}

func (p *CallbackPublisher) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		// 先收齐已受理消息的成功/失败结果，再关闭客户端，保留在途重试机会。
		p.pending.Wait()
		p.producer.AsyncClose()
		if err := <-p.done; err != nil {
			p.closeErr = fmt.Errorf("回调生产期间出现错误: %w", err)
		}
	})
	return p.closeErr
}
