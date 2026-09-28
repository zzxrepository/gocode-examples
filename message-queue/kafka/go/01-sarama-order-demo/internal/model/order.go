package model

import (
	"time"

	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/event"
)

type Order struct {
	ID          string `json:"order_id"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amount_cents"`
}

// OrderPayload 只描述订单业务，不重复事件的公共元数据。
type OrderPayload struct {
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amount_cents"`
}

// 别名用于缩短签名；公共格式由 event.Event 统一维护。
type OrderEvent = event.Event[OrderPayload]

func NewOrderEvent(id string, payload OrderPayload) OrderEvent {
	return OrderEvent{ID: id, Type: "order.status_changed", Version: 1, Source: "order-service", OccurredAt: time.Now().UTC(), Payload: payload}
}
