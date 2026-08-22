package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

// MySQLConfig 将连接信息集中在启动阶段，避免业务 SQL 关心 DSN 细节。
type MySQLConfig struct {
	User     string
	Password string
	Address  string
	Database string
}

// OpenMySQL 创建并验证可复用的 database/sql 连接池。
func OpenMySQL(ctx context.Context, cfg MySQLConfig) (*sql.DB, error) {
	driverCfg := mysql.NewConfig()
	driverCfg.User = cfg.User
	driverCfg.Passwd = cfg.Password
	driverCfg.Net = "tcp"
	driverCfg.Addr = cfg.Address
	driverCfg.DBName = cfg.Database

	// DATE/DATETIME 查询结果需要扫描到 time.Time，因此由驱动解析时间。
	driverCfg.ParseTime = true
	driverCfg.Loc = time.Local

	// 连接级超时：分别限制拨号、读取和写入网络 I/O。
	driverCfg.Timeout = 3 * time.Second
	driverCfg.ReadTimeout = 5 * time.Second
	driverCfg.WriteTimeout = 5 * time.Second

	// sql.Open 返回的是池句柄，通常并未立刻创建 TCP 连接。
	db, err := sql.Open("mysql", driverCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open MySQL handle: %w", err)
	}

	// 以下设置属于 database/sql 的连接池，不属于 MySQL 驱动。
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetConnMaxIdleTime(time.Minute)

	// PingContext 会借连接或新建连接，因此能验证地址、网络和认证信息。
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	return db, nil
}
