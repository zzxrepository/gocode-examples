package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// OpenDB 保留 GORM 默认版本探测和自动 Ping；返回时已访问数据库。
func OpenDB(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("MYSQL_DSN 未配置")
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("初始化 GORM：%w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接池：%w", err)
	}

	// 参数作用于底层连接池，不会预先创建这些数量的连接。
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// configuredDSN 优先使用文章中的 MYSQL_DSN；未设置时沿用前两个 Demo 的本机账号配置。
func configuredDSN() string {
	if dsn := os.Getenv("MYSQL_DSN"); dsn != "" {
		return dsn
	}
	database := os.Getenv("MYSQL_DATABASE")
	if database == "" {
		database = "gorm_demo"
	}
	cfg := mysqldriver.NewConfig()
	cfg.User = os.Getenv("MYSQL_USER")
	if cfg.User == "" {
		cfg.User = "root"
	}
	cfg.Passwd = os.Getenv("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = "127.0.0.1:3306"
	cfg.DBName = database
	cfg.ParseTime = true
	cfg.Loc = time.Local
	cfg.Timeout = 3 * time.Second
	cfg.ReadTimeout = 5 * time.Second
	cfg.WriteTimeout = 5 * time.Second
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg.FormatDSN()
}
