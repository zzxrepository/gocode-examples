package main

import (
	"fmt"
	"github.com/IBM/sarama"
	"strings"
)

const extraUsage = "lag"

func extra(cmd string, o options) error {
	if cmd != "lag" {
		return fmt.Errorf("unknown command: %s", cmd)
	}
	c, err := sarama.NewClient(strings.Split(o.brokers, ","), config())
	if err != nil {
		return err
	}
	defer c.Close()
	a, err := sarama.NewClusterAdminFromClient(c)
	if err != nil {
		return err
	}
	defer a.Close()
	parts, err := c.Partitions(o.topic)
	if err != nil {
		return err
	}
	offsets, err := a.ListConsumerGroupOffsets(o.group, map[string][]int32{o.topic: parts})
	if err != nil {
		return err
	}
	for _, p := range parts {
		end, err := c.GetOffset(o.topic, p, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		b := offsets.GetBlock(o.topic, p)
		if b == nil {
			return fmt.Errorf("missing partition %d", p)
		}
		if b.Err != sarama.ErrNoError {
			return b.Err
		}
		if b.Offset < 0 {
			fmt.Printf("partition=%d committed=none end=%d lag=unknown\n", p, end)
		} else {
			fmt.Printf("partition=%d committed=%d end=%d lag=%d\n", p, b.Offset, end, end-b.Offset)
		}
	}
	return nil
}
