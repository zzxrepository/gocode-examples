package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/messaging"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/model"
)

var (
	ErrInvalidOrder = errors.New("order_id 需为 1~64 位字母、数字、横线或下划线，amount_cents 必须大于 0")
	ErrNotFound     = errors.New("订单不存在")
	ErrConflict     = errors.New("订单已存在或状态不允许该操作")
	orderIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

type EventPublisher interface {
	Publish(context.Context, messaging.Record) error
}

type OrderService struct {
	mu        sync.Mutex
	orders    map[string]model.Order
	publisher EventPublisher
}

func NewOrderService(publisher EventPublisher) *OrderService {
	return &OrderService{orders: make(map[string]model.Order), publisher: publisher}
}

func (s *OrderService) Create(ctx context.Context, id string, amount int64) (model.Order, error) {
	if !orderIDPattern.MatchString(id) || amount <= 0 {
		return model.Order{}, ErrInvalidOrder
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.orders[id]; exists {
		return model.Order{}, ErrConflict
	}
	order := model.Order{ID: id, Status: "created", AmountCents: amount}
	return s.publishAndStore(ctx, order)
}

func (s *OrderService) Transition(ctx context.Context, id, status string) (model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, exists := s.orders[id]
	if !exists {
		return model.Order{}, ErrNotFound
	}
	if order.Status != "created" || (status != "paid" && status != "cancelled") {
		return model.Order{}, ErrConflict
	}
	order.Status = status
	return s.publishAndStore(ctx, order)
}

func (s *OrderService) Get(id string) (model.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.orders[id]
	if !ok {
		return model.Order{}, ErrNotFound
	}
	return order, nil
}

func (s *OrderService) publishAndStore(ctx context.Context, order model.Order) (model.Order, error) {
	// 实验期间持锁发布，保证本进程的状态检查与发送顺序；牺牲不同订单的并行能力。
	// 这不是数据库与 Kafka 的原子事务。重启丢失内存状态；线上应采用持久存储和 Outbox。
	event := model.OrderEvent{
		EventID: fmt.Sprintf("%s:%s", order.ID, order.Status), Version: 1,
		OrderID: order.ID, Status: order.Status, AmountCents: order.AmountCents,
	}
	value, err := json.Marshal(event)
	if err != nil {
		return model.Order{}, err
	}
	if err := s.publisher.Publish(ctx, messaging.Record{Key: order.ID, Value: value, EventID: event.EventID}); err != nil {
		return model.Order{}, err
	}
	s.orders[order.ID] = order
	return order, nil
}
