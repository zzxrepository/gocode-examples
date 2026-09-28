package controller

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/service"
)

type OrderController struct {
	orders   *service.OrderService
	accepted bool
}

func NewOrderController(orders *service.OrderService) *OrderController {
	return &OrderController{orders: orders}
}

func NewAsyncOrderController(orders *service.OrderService) *OrderController {
	return &OrderController{orders: orders, accepted: true}
}

func (c *OrderController) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		OrderID     string `json:"order_id"`
		AmountCents int64  `json:"amount_cents"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求必须是合法的订单 JSON，且不超过 4096 字节"})
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求只能包含一个 JSON 对象"})
		return
	}
	order, err := c.orders.Create(r.Context(), input.OrderID, input.AmountCents)
	if err != nil {
		respondError(w, err)
		return
	}
	if c.accepted {
		writeJSON(w, http.StatusAccepted, order)
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (c *OrderController) Pay(w http.ResponseWriter, r *http.Request) { c.transition(w, r, "paid") }
func (c *OrderController) Cancel(w http.ResponseWriter, r *http.Request) {
	c.transition(w, r, "cancelled")
}

func (c *OrderController) transition(w http.ResponseWriter, r *http.Request, status string) {
	order, err := c.orders.Transition(r.Context(), r.PathValue("id"), status)
	if err != nil {
		respondError(w, err)
		return
	}
	if c.accepted {
		writeJSON(w, http.StatusAccepted, order)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (c *OrderController) Get(w http.ResponseWriter, r *http.Request) {
	order, err := c.orders.Get(r.PathValue("id"))
	if err != nil {
		respondError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func respondError(w http.ResponseWriter, err error) {
	status := http.StatusServiceUnavailable
	message := "事件发布未确认，订单状态未更新；核查后用相同订单号重试"
	switch {
	case errors.Is(err, service.ErrInvalidOrder):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
		message = err.Error()
	default:
		log.Printf("order request failed: %v", err)
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
