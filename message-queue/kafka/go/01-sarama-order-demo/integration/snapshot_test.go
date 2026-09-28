package integration

import (
	"context"
	"errors"
	"fmt"

	"github.com/IBM/sarama"
)

// 按分区读回启动时已有的消息，完成后退出。只用于观察实验结果，不提交消费组 offset。
func readSnapshot(ctx context.Context, brokers []string, topic string, cfg *sarama.Config, visit func(*sarama.ConsumerMessage) error) (err error) {
	client, err := sarama.NewClient(brokers, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, client.Close()) }()
	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, consumer.Close()) }()
	partitions, err := client.Partitions(topic)
	if err != nil {
		return err
	}
	type window struct {
		partition  int32
		start, end int64
	}
	windows := make([]window, 0, len(partitions))
	for _, partition := range partitions {
		start, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
		if err != nil {
			return err
		}
		end, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		windows = append(windows, window{partition, start, end})
	}
	for _, w := range windows {
		if w.start == w.end {
			continue
		}
		pc, err := consumer.ConsumePartition(topic, w.partition, w.start)
		if err != nil {
			return err
		}
		err = func() (err error) {
			defer func() { err = errors.Join(err, pc.Close()) }()
			for {
				select {
				case <-ctx.Done():
					return fmt.Errorf("读取 partition=%d: %w", w.partition, ctx.Err())
				case failure, ok := <-pc.Errors():
					if !ok {
						return errors.New("消费者错误通道提前关闭")
					}
					return failure
				case msg, ok := <-pc.Messages():
					if !ok {
						return errors.New("消费者消息通道提前关闭")
					}
					if msg.Offset >= w.end {
						return nil
					}
					if err := visit(msg); err != nil {
						return err
					}
					if msg.Offset == w.end-1 {
						return nil
					}
				}
			}
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
