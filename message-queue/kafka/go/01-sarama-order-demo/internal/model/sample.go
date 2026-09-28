package model

import (
	"crypto/rand"
	"fmt"
)

// SampleEvents 为事务实验提供六条事件；每次创建新的事件 ID。
func SampleEvents() ([]OrderEvent, error) {
	var runID [8]byte
	if _, err := rand.Read(runID[:]); err != nil {
		return nil, err
	}
	payloads := []OrderPayload{
		{OrderID: "order-1001", Status: "created", AmountCents: 1200},
		{OrderID: "order-1002", Status: "created", AmountCents: 2300},
		{OrderID: "order-1003", Status: "created", AmountCents: 3500},
		{OrderID: "order-1001", Status: "paid", AmountCents: 1200},
		{OrderID: "order-1002", Status: "paid", AmountCents: 2300},
		{OrderID: "order-1003", Status: "cancelled", AmountCents: 3500},
	}
	events := make([]OrderEvent, 0, len(payloads))
	for _, payload := range payloads {
		id := fmt.Sprintf("%x-%s-%s", runID, payload.OrderID, payload.Status)
		events = append(events, NewOrderEvent(id, payload))
	}
	return events, nil
}
