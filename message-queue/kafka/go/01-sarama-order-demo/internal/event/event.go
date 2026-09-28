package event

import "time"

// Event 统一业务事件的公共信息；T 保留每种 Payload 的编译期类型。
// 这是项目自定义信封，不声称兼容完整的 CloudEvents 标准。
type Event[T any] struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Version    int       `json:"version"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurred_at"`
	Payload    T         `json:"payload"`
}
