package main

import (
	"fmt"
	"github.com/IBM/sarama"
	"sort"
	"strings"
)

const extraUsage = "snapshot|tombstone"

func extra(cmd string, o options) error {
	switch cmd {
	case "tombstone":
		p, err := sarama.NewSyncProducer(strings.Split(o.brokers, ","), config())
		if err != nil {
			return err
		}
		defer p.Close()
		_, _, err = p.SendMessage(&sarama.ProducerMessage{Topic: o.topic, Key: sarama.StringEncoder(o.prefix + "-1"), Value: nil})
		return err
	case "snapshot":
		state := map[string]Event{}
		err := consume(o, func(e Event, _ *sarama.ConsumerMessage) error {
			if e.Type == "Tombstone" {
				delete(state, e.OrderID)
			} else if e.OrderVersion >= state[e.OrderID].OrderVersion {
				state[e.OrderID] = e
			}
			return nil
		})
		if err != nil {
			return err
		}
		keys := make([]string, 0, len(state))
		for k := range state {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			e := state[k]
			fmt.Printf("STATE order=%s type=%s version=%d\n", k, e.Type, e.OrderVersion)
		}
		return nil
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
