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
	events := []OrderEvent{
		{OrderID: "order-1001", Status: "created", AmountCents: 1200},
		{OrderID: "order-1002", Status: "created", AmountCents: 2300},
		{OrderID: "order-1003", Status: "created", AmountCents: 3500},
		{OrderID: "order-1001", Status: "paid", AmountCents: 1200},
		{OrderID: "order-1002", Status: "paid", AmountCents: 2300},
		{OrderID: "order-1003", Status: "cancelled", AmountCents: 3500},
	}
	for i := range events {
		events[i].Version = 1
		events[i].EventID = fmt.Sprintf("%x-%s-%s", runID, events[i].OrderID, events[i].Status)
	}
	return events, nil
}
