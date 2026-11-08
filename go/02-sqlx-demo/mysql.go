package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

// openDatabase 先创建句柄，再通过 PingContext 建立/验证实际连接。
func openDatabase(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	// 当前 MySQL 驱动的 OpenConnector 只解析 DSN；此时尚未拨号。
	stats := db.Stats()
	log.Printf("Open 后连接池: open=%d idle=%d in_use=%d", stats.OpenConnections, stats.Idle, stats.InUse)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(time.Hour)
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	stats = db.Stats()
	log.Printf("Ping 后连接池: open=%d idle=%d in_use=%d", stats.OpenConnections, stats.Idle, stats.InUse)
	return db, nil
}

// localDSN 始终连接本机 TCP 3306；FormatDSN 负责正确组织账号与密码。
func localDSN(database string) string {
	cfg := mysql.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	if cfg.User == "" {
		cfg.User = "root"
	}
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"
	cfg.DBName = database
	cfg.Timeout = 3 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg.FormatDSN()
}
