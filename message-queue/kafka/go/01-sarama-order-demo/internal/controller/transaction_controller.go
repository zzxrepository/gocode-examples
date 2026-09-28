package controller

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/messaging"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/model"
)

type TransactionController struct {
	mu      sync.Mutex
	brokers []string
	topic   string
	cfg     *sarama.Config
}

func NewTransactionController(brokers []string, topic string, cfg *sarama.Config) *TransactionController {
	copy := *cfg
	copy.Producer.Transaction.ID = cfg.ClientID + "-http-txn"
	return &TransactionController{brokers: brokers, topic: topic, cfg: &copy}
}

func (c *TransactionController) Commit(w http.ResponseWriter, r *http.Request) {
	c.sendBatch(w, r, false)
}
func (c *TransactionController) Abort(w http.ResponseWriter, r *http.Request) {
	c.sendBatch(w, r, true)
}

func (c *TransactionController) sendBatch(w http.ResponseWriter, r *http.Request, abort bool) {
	// 本机单实例实验串行使用稳定事务 ID；多实例必须分配各自稳定且唯一的 ID。
	c.mu.Lock()
	defer c.mu.Unlock()
	events, err := model.SampleEvents()
	if err != nil {
		respondError(w, err)
		return
	}
	messages := make([]*sarama.ProducerMessage, 0, len(events))
	for _, event := range events {
		value, err := json.Marshal(event)
		if err != nil {
			respondError(w, err)
			return
		}
		messages = append(messages, messaging.NewMessage(c.topic, messaging.Record{Key: event.Payload.OrderID, Value: value, EventID: event.ID}))
	}
	if err := messaging.SendTransaction(r.Context(), c.brokers, c.cfg, messages, abort); err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"aborted": abort, "messages": len(messages)})
}
