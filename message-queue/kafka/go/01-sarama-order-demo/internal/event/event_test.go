package event

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDifferentPayloadsShareEnvelope(t *testing.T) {
	type UserPayload struct {
		UserID string `json:"user_id"`
	}
	original := Event[UserPayload]{ID: "user-event-1", Type: "user.registered", Version: 1, Source: "user-service", OccurredAt: time.Now().UTC(), Payload: UserPayload{UserID: "u-1"}}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	// 路由层只解析公共字段，业务 Handler 再解码自己的 Payload。
	var envelope Event[json.RawMessage]
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	var payload UserPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if envelope.Type != "user.registered" || envelope.ID != original.ID || payload.UserID != "u-1" {
		t.Fatal("事件信封或业务载荷丢失")
	}
}
