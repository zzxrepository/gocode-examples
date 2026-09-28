package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/controller"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/messaging"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/model"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/router"
	"github.com/zzxrepository/gocode-examples/message-queue/kafka/go/01-sarama-order-demo/internal/service"
)

func TestHTTPRoutesToKafka(t *testing.T) {
	brokers, topic, _ := integrationTopic(t)
	cfg := kafkaConfig()
	cfg.Producer.Compression = sarama.CompressionGZIP
	syncPublisher, err := messaging.NewSyncPublisher(brokers, topic, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer syncPublisher.Close()
	asyncPublisher, err := messaging.NewAsyncPublisher(brokers, topic, cfg)
	if err != nil {
		t.Fatal(err)
	}
	callbackPublisher, err := messaging.NewCallbackPublisher(brokers, topic, cfg)
	if err != nil {
		asyncPublisher.Close()
		t.Fatal(err)
	}
	defer func() {
		if callbackPublisher != nil {
			callbackPublisher.Close()
		}
		if asyncPublisher != nil {
			asyncPublisher.Close()
		}
	}()
	server := httptest.NewServer(router.New(
		controller.NewOrderController(service.NewOrderService(syncPublisher)),
		controller.NewAsyncOrderController(service.NewOrderService(asyncPublisher)),
		controller.NewAsyncOrderController(service.NewOrderService(callbackPublisher)),
		controller.NewTransactionController(brokers, topic, cfg),
	))
	defer server.Close()
	request := func(method, path, body string, want int) model.Order {
		t.Helper()
		req, err := http.NewRequest(method, server.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s got=%d want=%d body=%s", method, path, res.StatusCode, want, data)
		}
		var order model.Order
		if want < 300 {
			if err := json.Unmarshal(data, &order); err != nil {
				t.Fatal(err)
			}
		}
		return order
	}
	for _, scenario := range []string{"sync", "async", "callback"} {
		path := "/demo/" + scenario + "/orders"
		id := "http-" + scenario
		createStatus, updateStatus := http.StatusAccepted, http.StatusAccepted
		if scenario == "sync" {
			createStatus, updateStatus = http.StatusCreated, http.StatusOK
		}
		body := fmt.Sprintf(`{"order_id":%q,"amount_cents":1200}`, id)
		request("POST", path, body, createStatus)
		request("POST", path, body, http.StatusConflict)
		request("POST", path+"/"+id+"/pay", "", updateStatus)
		order := request("GET", path+"/"+id, "", http.StatusOK)
		if order.Status != "paid" {
			t.Fatal("支付后订单状态错误")
		}
		request("POST", path+"/"+id+"/cancel", "", http.StatusConflict)
		request("GET", path+"/missing", "", http.StatusNotFound)
		request("POST", path, `{"order_id":"bad","amount_cents":-1}`, http.StatusBadRequest)
		request("POST", path, `{"order_id":"bad","amount_cents":1} {}`, http.StatusBadRequest)
	}
	server.Close()
	if err := asyncPublisher.Close(); err != nil {
		t.Fatal(err)
	}
	if err := asyncPublisher.Close(); err != nil {
		t.Fatal("重复关闭应返回同一结果", err)
	}
	if err := asyncPublisher.Publish(context.Background(), messaging.Record{}); err == nil {
		t.Fatal("关闭后不能继续发布")
	}
	asyncPublisher = nil
	if err := callbackPublisher.Close(); err != nil {
		t.Fatal(err)
	}
	if err := callbackPublisher.Close(); err != nil {
		t.Fatal("重复关闭应返回同一结果", err)
	}
	if err := callbackPublisher.Publish(context.Background(), messaging.Record{}); err == nil {
		t.Fatal("关闭后不能继续发布")
	}
	callbackPublisher = nil
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	statuses := map[string][]string{}
	partitions := map[string]int32{}
	err = readSnapshot(ctx, brokers, topic, kafkaConfig(), func(msg *sarama.ConsumerMessage) error {
		var event model.OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			return err
		}
		if string(msg.Key) != event.Payload.OrderID || event.Payload.AmountCents != 1200 {
			return errors.New("消息内容错误")
		}
		if previous, ok := partitions[event.Payload.OrderID]; ok && previous != msg.Partition {
			return errors.New("同订单进入不同分区")
		}
		partitions[event.Payload.OrderID] = msg.Partition
		statuses[event.Payload.OrderID] = append(statuses[event.Payload.OrderID], event.Payload.Status)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 3 {
		t.Fatalf("期望三笔订单，实际 %+v", statuses)
	}
	for id, got := range statuses {
		if strings.Join(got, ",") != "created,paid" {
			t.Fatalf("%s 顺序或数量错误: %v", id, got)
		}
	}
}

func TestAsyncFailuresAreDrained(t *testing.T) {
	brokers, topic, _ := integrationTopic(t)
	cfg := kafkaConfig()
	cfg.Producer.MaxMessageBytes = 1
	for _, callback := range []bool{false, true} {
		var publisher interface {
			Publish(context.Context, messaging.Record) error
			Close() error
		}
		var err error
		if callback {
			publisher, err = messaging.NewCallbackPublisher(brokers, topic, cfg)
		} else {
			publisher, err = messaging.NewAsyncPublisher(brokers, topic, cfg)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := publisher.Publish(context.Background(), messaging.Record{EventID: "too-large", Key: "test", Value: []byte("oversized")}); err != nil {
			publisher.Close()
			t.Fatal(err)
		}
		err = publisher.Close()
		var configErr sarama.ConfigurationError
		if !errors.As(err, &configErr) || !strings.Contains(configErr.Error(), "MaxMessageBytes") {
			t.Fatalf("异步错误未返回: %v", err)
		}
	}
}
