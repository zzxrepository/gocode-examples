package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

// 将六条消息放在一个 Kafka 事务中；不包含数据库或支付接口的事务。
func SendTransaction(ctx context.Context, brokers []string, cfg *sarama.Config, messages []*sarama.ProducerMessage, abort bool) (err error) {
	producer, err := newTransactionProducer(ctx, brokers, cfg)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, producer.Close()) }()
	if err := producer.BeginTxn(); err != nil {
		return err
	}
	for _, msg := range messages {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, producer.AbortTxn())
		}
		if _, _, err := producer.SendMessage(msg); err != nil {
			// 致命错误不能继续复用生产者；可中止的事务尝试中止后退出。
			if producer.TxnStatus()&sarama.ProducerTxnFlagFatalError != 0 {
				return err
			}
			return errors.Join(err, producer.AbortTxn())
		}
	}
	if abort {
		if err := producer.AbortTxn(); err != nil {
			return err
		}
		fmt.Printf("txn aborted messages=%d\n", len(messages))
		return nil
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(err, producer.AbortTxn())
	}
	if err := producer.CommitTxn(); err != nil {
		// 提交错误可能意味着结果未知，不能自动把整批当作新事务重发。
		return fmt.Errorf("事务提交未确认，请核查事务状态，勿盲目重发: %w", err)
	}
	fmt.Printf("txn committed messages=%d\n", len(messages))
	return nil
}

func newTransactionProducer(ctx context.Context, brokers []string, cfg *sarama.Config) (sarama.SyncProducer, error) {
	// 相同事务 ID 刚结束事务时，协调者可能仍在收尾。只重试初始化，绝不重发业务批次。
	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		producer, err := sarama.NewSyncProducer(brokers, cfg)
		if !errors.Is(err, sarama.ErrConcurrentTransactions) || attempt == 9 {
			return producer, err
		}
		timer := time.NewTimer(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
