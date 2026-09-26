package main

import (
	"fmt"
	"github.com/IBM/sarama"
	"sort"
	"time"
)

const extraUsage = "aggregate"

func extra(cmd string, o options) error {
	if cmd != "aggregate" {
		return fmt.Errorf("unknown command: %s", cmd)
	}
	type total struct {
		count  int
		amount int64
	}
	windows := map[string]total{}
	// 有界回放实验：内存状态每次重建，必须使用新消费组从保留数据开头读取。
	err := consume(o, func(e Event, _ *sarama.ConsumerMessage) error {
		if e.Type != "OrderPaid" {
			return nil
		}
		t, err := time.Parse(time.RFC3339Nano, e.OccurredAt)
		if err != nil {
			return err
		}
		key := t.UTC().Truncate(time.Minute).Format(time.RFC3339)
		v := windows[key]
		v.count++
		v.amount += e.AmountCents
		windows[key] = v
		return nil
	})
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(windows))
	for k := range windows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := windows[k]
		fmt.Printf("WINDOW minute=%s count=%d amount_cents=%d\n", k, v.count, v.amount)
	}
	return nil
}
