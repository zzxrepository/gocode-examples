package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const demoTimeout = 30 * time.Second

func main() {
	initOnly := flag.Bool("init", false, "只迁移 users、products、orders 表")
	showSQL := flag.Bool("sql", false, "输出本次演示的 SQL")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), demoTimeout)
	defer cancel()
	if err := run(ctx, *initOnly, *showSQL); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, initOnly, showSQL bool) (err error) {
	db, err := OpenDB(configuredDSN())
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取连接池：%w", err)
	}
	defer func() { err = errors.Join(err, pool.Close()) }()
	if showSQL {
		db = db.Debug()
	}
	stats := pool.Stats()
	log.Printf("gorm.Open 后连接池: open=%d idle=%d in_use=%d",
		stats.OpenConnections, stats.Idle, stats.InUse)
	if err := migrateSchema(ctx, db); err != nil {
		return err
	}
	if initOnly {
		log.Println("AutoMigrate: users、products、orders 表已就绪")
		return nil
	}
	return runDemo(ctx, db)
}
