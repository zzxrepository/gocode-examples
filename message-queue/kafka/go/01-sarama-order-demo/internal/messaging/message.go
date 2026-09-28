package messaging

import "github.com/IBM/sarama"

// Record 是传输层输入；不依赖订单、支付或用户等业务结构体。
type Record struct {
	Key     string
	Value   []byte
	EventID string
}

func NewMessage(topic string, record Record) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder(record.Key), Value: sarama.ByteEncoder(record.Value), Metadata: record.EventID}
}
