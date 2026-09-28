package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

func integrationTopic(t *testing.T) ([]string, string, sarama.ClusterAdmin) {
	t.Helper()
	address := os.Getenv("KAFKA_TEST_BROKER")
	if address == "" {
		t.Skip("设置 KAFKA_TEST_BROKER 后运行真实 Kafka 测试")
	}
	brokers := []string{address}
	admin, err := sarama.NewClusterAdmin(brokers, kafkaConfig())
	if err != nil {
		t.Fatal(err)
	}
	topic := fmt.Sprintf("order-test-%d", time.Now().UnixNano())
	if err := admin.CreateTopic(topic, &sarama.TopicDetail{NumPartitions: 3, ReplicationFactor: 1}, false); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.DeleteTopic(topic); err != nil {
			t.Error(err)
		}
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	return brokers, topic, admin
}

func collectGroup(t *testing.T, brokers []string, topic, group string, cfg *sarama.Config, expected int) []OrderEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	var mu sync.Mutex
	var events []OrderEvent
	received := make(chan struct{}, 64)
	done := make(chan error, 1)
	go func() {
		done <- consumeGroup(ctx, brokers, topic, group, cfg, func(_ context.Context, msg *sarama.ConsumerMessage) error {
			var event OrderEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				return err
			}
			mu.Lock()
			events = append(events, event)
			mu.Unlock()
			received <- struct{}{}
			return nil
		})
	}()
	for i := 0; i < expected; i++ {
		select {
		case <-received:
		case err := <-done:
			t.Fatalf("消费提前退出: %v", err)
		case <-ctx.Done():
			cancel()
			<-done
			t.Fatal("未在期限内收到预期消息")
		}
	}
	// 等待当前提交并留出观察多余记录的窗口；消费完成以消息数判断，不以启动后固定时长判断。
	select {
	case <-time.After(time.Second):
	case <-ctx.Done():
	}
	cancel()
	err := <-done
	if err != nil {
		t.Fatal(err)
	}
	return events
}

func TestGroupResumeAndIndependentGroups(t *testing.T) {
	brokers, topic, admin := integrationTopic(t)
	group := topic + "-progress"
	other := topic + "-audit"
	defer admin.DeleteConsumerGroup(group)
	defer admin.DeleteConsumerGroup(other)
	messages, err := orderMessages(topic)
	if err != nil {
		t.Fatal(err)
	}
	if err := sendSync(context.Background(), brokers, kafkaConfig(), messages); err != nil {
		t.Fatal(err)
	}
	if got := collectGroup(t, brokers, topic, group, kafkaConfig(), 6); len(got) != 6 {
		t.Fatalf("首轮收到 %d 条", len(got))
	}
	// 用协调者返回的已提交位置验证，而不把“处理日志出现了”误当作提交成功。
	offsets, err := admin.ListConsumerGroupOffsets(group, map[string][]int32{topic: {0, 1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	want := map[int32]int64{}
	for _, msg := range messages {
		want[msg.Partition] = msg.Offset + 1
	}
	for partition, next := range want {
		block := offsets.GetBlock(topic, partition)
		if block == nil || block.Err != sarama.ErrNoError || block.Offset != next {
			t.Fatalf("partition=%d 提交位置错误: %+v，期望 %d", partition, block, next)
		}
	}
	more, err := orderMessages(topic)
	if err != nil {
		t.Fatal(err)
	}
	if err := sendSync(context.Background(), brokers, kafkaConfig(), more[:1]); err != nil {
		t.Fatal(err)
	}
	got := collectGroup(t, brokers, topic, group, kafkaConfig(), 1)
	if len(got) != 1 || got[0].EventID != more[0].Metadata.(OrderEvent).EventID {
		t.Fatalf("重启应只收到新增消息，实际 %+v", got)
	}
	if got := collectGroup(t, brokers, topic, other, kafkaConfig(), 7); len(got) != 7 {
		t.Fatalf("独立组应收到七条，实际 %d", len(got))
	}
}

func TestGroupFailureDoesNotSkipRecord(t *testing.T) {
	brokers, topic, admin := integrationTopic(t)
	group := topic + "-failure"
	defer admin.DeleteConsumerGroup(group)
	messages, err := orderMessages(topic)
	if err != nil {
		t.Fatal(err)
	}
	cfg := kafkaConfig()
	cfg.Producer.Partitioner = sarama.NewManualPartitioner
	// 所有消息放分区 0，使首次 paid 的 offset 可预测为 3。
	for _, msg := range messages {
		msg.Partition = 0
	}
	if err := sendSync(context.Background(), brokers, cfg, messages); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = consumeGroup(ctx, brokers, topic, group, kafkaConfig(), orderHandler(group, "test", 0, "paid"))
	if err == nil || !strings.Contains(err.Error(), "模拟 paid 处理失败") {
		t.Fatalf("应报告业务错误: %v", err)
	}
	offsets, err := admin.ListConsumerGroupOffsets(group, map[string][]int32{topic: {0}})
	if err != nil {
		t.Fatal(err)
	}
	block := offsets.GetBlock(topic, 0)
	if block == nil || block.Err != sarama.ErrNoError || block.Offset != 3 {
		t.Fatalf("失败消息被跳过或成功消息未提交: %+v", block)
	}
	got := collectGroup(t, brokers, topic, group, kafkaConfig(), 3)
	if len(got) != 3 || got[0].Status != "paid" {
		t.Fatalf("应从失败消息恢复，实际 %+v", got)
	}
}

func TestTransactionVisibility(t *testing.T) {
	brokers, topic, admin := integrationTopic(t)
	aborted, err := orderMessages(topic)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := orderMessages(topic)
	if err != nil {
		t.Fatal(err)
	}
	for i, batch := range [][]*sarama.ProducerMessage{aborted, committed} {
		cfg := kafkaConfig()
		cfg.Producer.Transaction.ID = topic + "-writer"
		if err := sendTransaction(context.Background(), brokers, cfg, batch, i == 0); err != nil {
			t.Fatal(err)
		}
	}
	for _, committedOnly := range []bool{true, false} {
		group := fmt.Sprintf("%s-%t", topic, committedOnly)
		defer admin.DeleteConsumerGroup(group)
		cfg := kafkaConfig()
		cfg.Consumer.Offsets.AutoCommit.Enable = true // 同时覆盖自动提交已标记位移。
		want := 6
		if !committedOnly {
			cfg.Consumer.IsolationLevel = sarama.ReadUncommitted
			want = 12
		}
		got := collectGroup(t, brokers, topic, group, cfg, want)
		if len(got) != want {
			t.Fatalf("committedOnly=%t 收到 %d 条，期望 %d", committedOnly, len(got), want)
		}
		ids := map[string]bool{}
		for _, event := range got {
			ids[event.EventID] = true
		}
		for _, msg := range committed {
			if !ids[msg.Metadata.(OrderEvent).EventID] {
				t.Fatal("已提交事件缺失")
			}
		}
		for _, msg := range aborted {
			if ids[msg.Metadata.(OrderEvent).EventID] == committedOnly {
				t.Fatal("中止事件的隔离可见性错误")
			}
		}
	}
}
