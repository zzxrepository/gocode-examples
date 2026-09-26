package main

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/go-sql-driver/mysql"
	"os"
	"strings"
	"time"
)

const extraUsage = "project|transaction|place|relay"

func extra(cmd string, o options) error {
	switch cmd {
	case "place", "relay":
		return outbox(cmd, o)
	case "transaction":
		return transaction(o)
	case "project":
		return project(o)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
func transaction(o options) error {
	c := config()
	c.Producer.Transaction.ID = env("TRANSACTIONAL_ID", o.topic+"-go-producer")
	// 前一个实例结束事务后，协调器可能尚在完成标记；仅对该瞬态错误有限重试。
	var p sarama.SyncProducer
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for {
		p, err = sarama.NewSyncProducer(strings.Split(o.brokers, ","), c)
		if !errors.Is(err, sarama.ErrConcurrentTransactions) || time.Now().After(deadline) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return err
	}
	defer p.Close()
	if err = p.BeginTxn(); err != nil {
		return err
	}
	for _, e := range events(o) {
		m, err := message(o.topic, e)
		if err != nil {
			_ = p.AbortTxn()
			return err
		}
		if _, _, err = p.SendMessage(m); err != nil {
			_ = p.AbortTxn()
			return err
		}
	}
	if o.abort {
		err = p.AbortTxn()
		fmt.Println("ABORTED")
	} else {
		err = p.CommitTxn()
		fmt.Println("COMMITTED")
	}
	return err
}
func project(o options) error {
	// 配置来自环境，避免把账户口令写入源码。
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		return errors.New("set MYSQL_DSN; run schema.sql first")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		return err
	}
	return consume(o, func(e Event, _ *sarama.ConsumerMessage) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		_, err = tx.Exec("INSERT INTO processed_events(consumer_name,event_id) VALUES (?,?)", o.group, e.EventID)
		if err != nil {
			var me *mysql.MySQLError
			if errors.As(err, &me) && me.Number == 1062 {
				fmt.Println("DUPLICATE", e.EventID)
				return nil
			}
			return err
		}
		_, err = tx.Exec(`INSERT INTO order_projection(consumer_name,order_id,status,version,amount_cents) VALUES (?,?,?,?,?)
   ON DUPLICATE KEY UPDATE status=IF(VALUES(version)>version,VALUES(status),status),amount_cents=IF(VALUES(version)>version,VALUES(amount_cents),amount_cents),version=GREATEST(version,VALUES(version))`, o.group, e.OrderID, e.Type, e.OrderVersion, e.AmountCents)
		if err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		// 模拟数据库提交成功、Kafka 位点未提交的失败窗口。重启后应由唯一键去重。
		if os.Getenv("FAIL_AFTER_DB") == "1" {
			return errors.New("injected failure after DB commit; offset not committed")
		}
		fmt.Println("APPLIED", e.EventID)
		return nil
	})
}
