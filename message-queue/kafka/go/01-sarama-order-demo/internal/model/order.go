package model

type Order struct {
	ID          string `json:"order_id"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amount_cents"`
}

type OrderEvent struct {
	EventID     string `json:"event_id"`
	Version     int    `json:"version"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	AmountCents int64  `json:"amount_cents"`
}
