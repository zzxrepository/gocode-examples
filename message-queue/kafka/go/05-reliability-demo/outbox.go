package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/IBM/sarama"
	"os"
	"strings"
)

func outbox(cmd string, o options) error {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		return fmt.Errorf("set MYSQL_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if cmd == "place" {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		for _, e := range events(o) {
			if e.OrderVersion == 1 {
				if _, err = tx.Exec("INSERT INTO source_orders(order_id,status) VALUES (?,?)", e.OrderID, e.Type); err != nil {
					return err
				}
			} else {
				if _, err = tx.Exec("UPDATE source_orders SET status=? WHERE order_id=?", e.Type, e.OrderID); err != nil {
					return err
				}
			}
			b, err := eventBytes(e)
			if err != nil {
				return err
			}
			if _, err = tx.Exec("INSERT INTO order_outbox(event_id,topic,payload) VALUES (?,?,?)", e.EventID, o.topic, b); err != nil {
				return err
			}
		}
		if err = tx.Commit(); err == nil {
			fmt.Println("OUTBOX_SAVED", 2*o.orders)
		}
		return err
	}
	// 单投递进程演示；先取得快照再发送，发送成功后才标记。重复发送由消费端去重。
	rows, err := db.Query("SELECT event_id,payload FROM order_outbox WHERE topic=? AND published=0 ORDER BY id LIMIT 100", o.topic)
	if err != nil {
		return err
	}
	var pending []Event
	for rows.Next() {
		var id string
		var b []byte
		if err = rows.Scan(&id, &b); err != nil {
			rows.Close()
			return err
		}
		var e Event
		if err = json.Unmarshal(b, &e); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	p, err := sarama.NewSyncProducer(strings.Split(o.brokers, ","), config())
	if err != nil {
		return err
	}
	defer p.Close()
	for _, e := range pending {
		m, err := message(o.topic, e)
		if err != nil {
			return err
		}
		if _, _, err = p.SendMessage(m); err != nil {
			return err
		}
		if os.Getenv("FAIL_AFTER_SEND") == "1" {
			return fmt.Errorf("injected failure after Kafka ACK; outbox remains pending")
		}
		if _, err = db.Exec("UPDATE order_outbox SET published=1 WHERE event_id=?", e.EventID); err != nil {
			return err
		}
		fmt.Println("RELAYED", e.EventID)
	}
	return nil
}
