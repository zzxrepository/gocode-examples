package messaging

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
)

// AsyncPublisher 演示调用方只等待入队的异步发送；HTTP 只能报告已受理。
type AsyncPublisher struct {
	producer  sarama.AsyncProducer
	topic     string
	done      chan error
	mu        sync.RWMutex
	closed    bool
	pending   sync.WaitGroup
	closeOnce sync.Once
	closeErr  error
}

func NewAsyncPublisher(brokers []string, topic string, cfg *sarama.Config) (*AsyncPublisher, error) {
	copy := *cfg
	copy.Producer.Return.Successes = true
	copy.Producer.Return.Errors = true
	producer, err := sarama.NewAsyncProducer(brokers, &copy)
	if err != nil {
		return nil, err
	}
	p := &AsyncPublisher{producer: producer, topic: topic, done: make(chan error, 1)}
	go func() {
		var firstErr error
		successes, failures := producer.Successes(), producer.Errors()
		for successes != nil || failures != nil {
			select {
			case _, ok := <-successes:
				if !ok {
					successes = nil
					continue
				}
				// 成功结果只用于确认完成，不向 HTTP 调用方逐条回传。
				p.pending.Done()
			case failure, ok := <-failures:
				if !ok {
					failures = nil
					continue
				}
				log.Printf("async failed key=%v err=%v", failure.Msg.Key, failure.Err)
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

func (p *AsyncPublisher) Publish(ctx context.Context, record Record) error {
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
		log.Printf("async queued event=%s", record.EventID)
		return nil // 仅进入客户端队列，不代表 Broker 已确认。
	}
}

func (p *AsyncPublisher) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		// 先收齐已受理消息的成功/失败结果，再关闭客户端，保留在途重试机会。
		p.pending.Wait()
		p.producer.AsyncClose()
		if err := <-p.done; err != nil {
			p.closeErr = fmt.Errorf("异步生产期间出现错误: %w", err)
		}
	})
	return p.closeErr
}
