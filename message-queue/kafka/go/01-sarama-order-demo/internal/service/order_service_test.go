package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/messaging"
)

type failingPublisher struct {
	failure error
	events  []messaging.Record
}

func (p *failingPublisher) Publish(_ context.Context, e messaging.Record) error {
	p.events = append(p.events, e)
	return p.failure
}

func TestFailedPublishDoesNotAdvanceOrder(t *testing.T) {
	p := &failingPublisher{failure: errors.New("broker unavailable")}
	s := NewOrderService(p)
	ctx := context.Background()
	if _, err := s.Create(ctx, "test", 1200); err == nil {
		t.Fatal("应报告失败")
	}
	if _, err := s.Get("test"); !errors.Is(err, ErrNotFound) {
		t.Fatal("发送失败却保存了订单")
	}
	p.failure = nil
	if _, err := s.Create(ctx, "test", 1200); err != nil {
		t.Fatal(err)
	}
	if p.events[0].EventID != p.events[1].EventID {
		t.Fatal("重试必须复用业务事件 ID")
	}
	p.failure = errors.New("timeout")
	if _, err := s.Transition(ctx, "test", "paid"); err == nil {
		t.Fatal("应报告失败")
	}
	order, _ := s.Get("test")
	if order.Status != "created" {
		t.Fatal("发送失败却推进了状态")
	}
}
