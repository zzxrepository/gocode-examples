package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

// Event 是两个语言版本共用的协议；金额使用整数分，避免浮点误差。
type Event struct {
	SchemaVersion int    `json:"schema_version"`
	EventID       string `json:"event_id"`
	OrderID       string `json:"order_id"`
	Type          string `json:"type"`
	OrderVersion  int    `json:"order_version"`
	AmountCents   int64  `json:"amount_cents"`
	OccurredAt    string `json:"occurred_at"`
}

func (e Event) validate() error {
	if e.SchemaVersion != 1 || e.EventID == "" || e.OrderID == "" || e.AmountCents < 0 {
		return errors.New("invalid event identity/schema/amount")
	}
	if !((e.Type == "OrderCreated" && e.OrderVersion == 1) || ((e.Type == "OrderPaid" || e.Type == "OrderCancelled") && e.OrderVersion == 2)) {
		return errors.New("invalid event type/version")
	}
	_, err := time.Parse(time.RFC3339Nano, e.OccurredAt)
	return err
}

type options struct {
	brokers, topic, group, prefix, mode string
	orders, max, partitions, replicas   int
	duplicate, abort                    bool
	timeout, delay                      time.Duration
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func config() *sarama.Config {
	c := sarama.NewConfig()
	// 协议能力设为 Sarama 已支持的 4.0；Broker 为 4.3.1，不虚构客户端版本常量。
	c.Version = sarama.V4_0_0_0
	c.ClientID = "order-course-go"
	c.Net.DialTimeout = 10 * time.Second
	c.Net.ReadTimeout = 10 * time.Second
	c.Net.WriteTimeout = 10 * time.Second
	c.Producer.RequiredAcks = sarama.WaitForAll
	c.Producer.Idempotent = true
	c.Net.MaxOpenRequests = 1 // Sarama 幂等生产者的约束，与 Java 客户端上限不同。
	c.Producer.Retry.Max = 5
	c.Producer.Return.Successes = true
	c.Producer.Compression = sarama.CompressionSnappy
	c.Producer.Partitioner = func(string) sarama.Partitioner { return crcPartitioner{} }
	c.Consumer.Offsets.Initial = sarama.OffsetOldest
	c.Consumer.Offsets.AutoCommit.Enable = false
	c.Consumer.IsolationLevel = sarama.ReadCommitted
	c.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}
	c.Admin.Timeout = 10 * time.Second
	return c
}

type crcPartitioner struct{}

func (crcPartitioner) Partition(m *sarama.ProducerMessage, n int32) (int32, error) {
	b, err := m.Key.Encode()
	if err != nil {
		return 0, err
	}
	return int32(crc32.ChecksumIEEE(b) % uint32(n)), nil
}
func (crcPartitioner) RequiresConsistency() bool { return true }
func eventBytes(e Event) ([]byte, error) {
	if err := e.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(e)
}
func events(o options) []Event {
	result := make([]Event, 0, 2*o.orders)
	for i := 1; i <= o.orders; i++ {
		id := o.prefix + "-" + strconv.Itoa(i)
		for version, kind := range []string{"OrderCreated", "OrderPaid"} {
			e := Event{1, id + "-" + strconv.Itoa(version+1), id, kind, version + 1, 1000, time.Now().UTC().Format(time.RFC3339Nano)}
			result = append(result, e)
			if o.duplicate {
				result = append(result, e)
			}
		}
	}
	return result
}
func message(topic string, e Event) (*sarama.ProducerMessage, error) {
	b, err := eventBytes(e)
	if err != nil {
		return nil, err
	}
	return &sarama.ProducerMessage{Topic: topic, Key: sarama.StringEncoder(e.OrderID), Value: sarama.ByteEncoder(b)}, nil
}
func createTopic(o options, compact bool) error {
	a, err := sarama.NewClusterAdmin(strings.Split(o.brokers, ","), config())
	if err != nil {
		return err
	}
	defer a.Close()
	entries := map[string]*string{}
	if compact {
		v := "compact"
		entries["cleanup.policy"] = &v
	}
	err = a.CreateTopic(o.topic, &sarama.TopicDetail{NumPartitions: int32(o.partitions), ReplicationFactor: int16(o.replicas), ConfigEntries: entries}, false)
	if errors.Is(err, sarama.ErrTopicAlreadyExists) {
		fmt.Println("topic already exists; existing settings unchanged")
		return nil
	}
	if err == nil {
		fmt.Println("created", o.topic)
	}
	return err
}
func produce(o options, async bool) error {
	if !async {
		p, err := sarama.NewSyncProducer(strings.Split(o.brokers, ","), config())
		if err != nil {
			return err
		}
		defer p.Close()
		for _, e := range events(o) {
			m, err := message(o.topic, e)
			if err != nil {
				return err
			}
			part, offset, err := p.SendMessage(m)
			if err != nil {
				return err
			}
			fmt.Printf("ACK event=%s partition=%d offset=%d\n", e.EventID, part, offset)
		}
		return nil
	}
	p, err := sarama.NewAsyncProducer(strings.Split(o.brokers, ","), config())
	if err != nil {
		return err
	}
	var acknowledgements sync.WaitGroup
	done := make(chan error, 1)
	// 同时排空成功/失败通道；只写 Input 而不读响应可能阻塞生产者。
	go func() {
		var first error
		success, fail := p.Successes(), p.Errors()
		for success != nil || fail != nil {
			select {
			case m, ok := <-success:
				if !ok {
					success = nil
				} else {
					acknowledgements.Done()
					fmt.Printf("ACK partition=%d offset=%d\n", m.Partition, m.Offset)
				}
			case e, ok := <-fail:
				if !ok {
					fail = nil
				} else {
					acknowledgements.Done()
					if first == nil {
						first = e.Err
					}
				}
			}
		}
		done <- first
	}()
	for _, e := range events(o) {
		m, err := message(o.topic, e)
		if err != nil {
			p.AsyncClose()
			<-done
			return err
		}
		acknowledgements.Add(1)
		p.Input() <- m
	}
	// 等待每条消息的结果后再关闭；响应通道由后台协程持续排空。
	acknowledgements.Wait()
	p.AsyncClose()
	return <-done
}

type handler struct {
	o       options
	process func(Event, *sarama.ConsumerMessage) error
	cancel  context.CancelFunc
	mu      sync.Mutex
	count   int
	err     error
}

func (h *handler) Setup(s sarama.ConsumerGroupSession) error {
	fmt.Printf("ASSIGNED %v\n", s.Claims())
	return nil
}
func (h *handler) Cleanup(s sarama.ConsumerGroupSession) error {
	fmt.Printf("REVOKED %v\n", s.Claims())
	return nil
}
func (h *handler) ConsumeClaim(s sarama.ConsumerGroupSession, c sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-s.Context().Done():
			return nil
		case m, ok := <-c.Messages():
			if !ok {
				return nil
			}
			// 教学实现串行执行业务处理，避免跨分区并发更新实验状态。
			h.mu.Lock()
			if h.err != nil || (h.o.max > 0 && h.count >= h.o.max) {
				h.mu.Unlock()
				return nil
			}
			var e Event
			err := json.Unmarshal(m.Value, &e)
			if err == nil {
				err = e.validate()
			}
			if m.Value == nil && compactTopic {
				e = Event{OrderID: string(m.Key), Type: "Tombstone"}
				err = nil
			}
			if err == nil && h.o.delay > 0 {
				select {
				case <-time.After(h.o.delay):
				case <-s.Context().Done():
					h.mu.Unlock()
					return nil
				}
			}
			if err == nil {
				err = h.process(e, m)
			}
			if err != nil {
				h.err = err
				h.cancel()
				h.mu.Unlock()
				return err
			}
			// 标记的是下一条 Offset；只有业务成功后才提交，失败消息保持未提交。
			s.MarkMessage(m, "")
			s.Commit()
			h.count++
			if h.o.max > 0 && h.count >= h.o.max {
				h.cancel()
			}
			h.mu.Unlock()
		}
	}
}
func consume(o options, process func(Event, *sarama.ConsumerMessage) error) error {
	g, err := sarama.NewConsumerGroup(strings.Split(o.brokers, ","), o.group, config())
	if err != nil {
		return err
	}
	defer g.Close()
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, o.timeout)
	defer cancel()
	h := &handler{o: o, process: process, cancel: cancel}
	for ctx.Err() == nil {
		if err = g.Consume(ctx, []string{o.topic}, h); err != nil {
			break
		}
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.err != nil {
		return h.err
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	if o.max > 0 && h.count < o.max {
		return fmt.Errorf("received %d/%d messages before stop/timeout", h.count, o.max)
	}
	fmt.Printf("CONSUMED %d\n", h.count)
	return nil
}
func printEvent(e Event, m *sarama.ConsumerMessage) error {
	b, _ := json.Marshal(e)
	fmt.Printf("EVENT partition=%d offset=%d %s\n", m.Partition, m.Offset, b)
	return nil
}
func inspect(o options) error {
	c, err := sarama.NewClient(strings.Split(o.brokers, ","), config())
	if err != nil {
		return err
	}
	defer c.Close()
	parts, err := c.Partitions(o.topic)
	if err != nil {
		return err
	}
	for _, p := range parts {
		leader, err := c.Leader(o.topic, p)
		if err != nil {
			return err
		}
		rep, err := c.Replicas(o.topic, p)
		if err != nil {
			return err
		}
		isr, err := c.InSyncReplicas(o.topic, p)
		if err != nil {
			return err
		}
		first, err := c.GetOffset(o.topic, p, sarama.OffsetOldest)
		if err != nil {
			return err
		}
		end, err := c.GetOffset(o.topic, p, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		fmt.Printf("partition=%d leader=%d replicas=%v isr=%v start=%d end=%d\n", p, leader.ID(), rep, isr, first, end)
	}
	return nil
}
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: demo init|produce|consume|inspect|"+extraUsage)
		os.Exit(2)
	}
	o := options{}
	fs := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	fs.StringVar(&o.brokers, "brokers", env("KAFKA_BROKERS", "127.0.0.1:9092"), "bootstrap servers")
	fs.StringVar(&o.topic, "topic", env("KAFKA_TOPIC", defaultTopic), "topic")
	fs.StringVar(&o.group, "group", env("KAFKA_GROUP", defaultTopic+"-go"), "consumer group")
	fs.StringVar(&o.prefix, "prefix", "order", "order ID prefix")
	fs.StringVar(&o.mode, "mode", "sync", "sync or async")
	fs.IntVar(&o.orders, "orders", 3, "orders, two events each")
	fs.IntVar(&o.max, "max", 6, "records to consume; 0 until timeout")
	fs.IntVar(&o.partitions, "partitions", 3, "topic partitions")
	fs.IntVar(&o.replicas, "replicas", 1, "replication factor")
	fs.BoolVar(&o.duplicate, "duplicate", false, "send each business event twice")
	fs.BoolVar(&o.abort, "abort", false, "abort transaction")
	fs.DurationVar(&o.timeout, "timeout", 30*time.Second, "consumer deadline")
	fs.DurationVar(&o.delay, "delay", 0, "processing delay per record")
	fs.Parse(os.Args[2:])
	if o.orders < 1 || o.max < 0 || o.partitions < 1 || o.replicas < 1 || o.timeout <= 0 || o.delay < 0 || (o.mode != "sync" && o.mode != "async") {
		fmt.Fprintln(os.Stderr, "invalid option")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "init":
		err = createTopic(o, compactTopic)
	case "produce":
		err = produce(o, o.mode == "async")
	case "consume":
		err = consume(o, printEvent)
	case "inspect":
		err = inspect(o)
	default:
		err = extra(os.Args[1], o)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}
