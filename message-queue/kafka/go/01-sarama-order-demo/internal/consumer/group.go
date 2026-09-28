package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/model"
)

type MessageHandler func(context.Context, *sarama.ConsumerMessage) error

type orderGroupHandler struct {
	handle       MessageHandler
	fail         func(error)
	manualCommit bool
}

func (h *orderGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	fmt.Printf("assigned member=%s claims=%v\n", session.MemberID(), session.Claims())
	return nil
}

func (h *orderGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	fmt.Printf("released member=%s claims=%v\n", session.MemberID(), session.Claims())
	return nil
}

func (h *orderGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Sarama 已经为每个 claim 启动 goroutine；此处顺序处理，不能再次 fire-and-forget。
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if session.Context().Err() != nil {
				return nil
			}
			if err := h.handle(session.Context(), msg); err != nil {
				if session.Context().Err() != nil {
					return nil
				}
				h.fail(fmt.Errorf("处理 partition=%d offset=%d: %w", msg.Partition, msg.Offset, err))
				return err // 不标记失败消息，不继续消费该分区后续消息。
			}
			// MarkMessage 标记“下一条” offset，即 msg.Offset+1；它本身不是提交。
			session.MarkMessage(msg, "")
			if h.manualCommit {
				session.Commit() // 没有 error 返回值；提交错误从 group.Errors() 收集。
			}
		}
	}
}

func RunGroup(parent context.Context, brokers []string, topic, groupID string, cfg *sarama.Config, handle MessageHandler) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return fmt.Errorf("创建消费者组: %w", err)
	}
	// 保存首个错误，并取消整个实例，避免失败后继续推进对应分区位移。
	failures := make(chan error, 1)
	fail := func(err error) {
		select {
		case failures <- err:
		default:
		}
		cancel()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for err := range group.Errors() {
			fail(err)
		}
	}()
	handler := &orderGroupHandler{handle: handle, fail: fail, manualCommit: !cfg.Consumer.Offsets.AutoCommit.Enable}
	for ctx.Err() == nil {
		// 每次 Consume 对应一次 session；再均衡结束后必须重新进入。
		if err := group.Consume(ctx, []string{topic}, handler); err != nil {
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				fail(err)
			}
			break
		}
	}
	closeErr := group.Close()
	<-done // Close 关闭 Errors 通道，等待收集完最后的提交错误。
	select {
	case err := <-failures:
		return errors.Join(err, closeErr)
	default:
		return closeErr
	}
}

func OrderHandler(group, clientID string, delay time.Duration, failStatus string) MessageHandler {
	return func(ctx context.Context, msg *sarama.ConsumerMessage) error {
		var event model.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return fmt.Errorf("JSON 解码: %w", err)
		}
		if event.Type != "order.status_changed" || event.Source == "" || event.OccurredAt.IsZero() || event.Version != 1 || event.ID == "" || event.Payload.OrderID == "" || string(msg.Key) != event.Payload.OrderID || event.Payload.AmountCents <= 0 {
			return fmt.Errorf("不支持的订单事件: %+v", event)
		}
		switch event.Payload.Status {
		case "created", "paid", "cancelled":
		default:
			return fmt.Errorf("未知状态 %q", event.Payload.Status)
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
			}
		}
		if event.Payload.Status == failStatus {
			return fmt.Errorf("模拟 %s 处理失败", failStatus)
		}
		// 业务动作仅打印；真实数据库更新、去重必须在返回 nil 之前完成。
		fmt.Printf("handled group=%s client=%s partition=%d offset=%d event=%s order=%s status=%s amount=%d\n",
			group, clientID, msg.Partition, msg.Offset, event.ID, event.Payload.OrderID, event.Payload.Status, event.Payload.AmountCents)
		return nil
	}
}
