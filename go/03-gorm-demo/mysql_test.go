package main

import (
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestConfiguredDSN(t *testing.T) {
	tests := []struct {
		name, user, database, wantUser, wantDB string
	}{
		{"defaults", "", "", "root", "gorm_demo"},
		{"environment", "demo_user", "local_demo", "demo_user", "local_demo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MYSQL_DSN", "")
			t.Setenv("MYSQL_USER", tt.user)
			t.Setenv("MYSQL_DATABASE", tt.database)
			t.Setenv("MYSQL_PASSWORD", "example-password")
			cfg, err := mysqldriver.ParseDSN(configuredDSN())
			if err != nil {
				t.Fatal(err)
			}
			if cfg.User != tt.wantUser || cfg.DBName != tt.wantDB || cfg.Addr != "127.0.0.1:3306" || cfg.Net != "tcp" {
				t.Fatal("DSN 未采用预期的本机配置")
			}
			if !cfg.ParseTime || cfg.Passwd != "example-password" || cfg.Timeout == 0 || cfg.ReadTimeout == 0 || cfg.WriteTimeout == 0 {
				t.Fatal("时间解析、密码或网络超时配置错误")
			}
		})
	}
	t.Run("explicit DSN", func(t *testing.T) {
		const dsn = "demo:example@tcp(127.0.0.1:3306)/gorm_demo?parseTime=true"
		t.Setenv("MYSQL_DSN", dsn)
		if configuredDSN() != dsn {
			t.Fatal("MYSQL_DSN 应优先于分项配置")
		}
	})
}

func TestOpenDBInvalidDSN(t *testing.T) {
	for _, dsn := range []string{"", "invalid-dsn"} {
		t.Run(dsn, func(t *testing.T) {
			if _, err := OpenDB(dsn); err == nil {
				t.Fatal("无效 DSN 应返回错误")
			}
		})
	}
}

func TestCreateOrderInvalidInput(t *testing.T) {
	tests := []struct {
		name              string
		userID, productID uint
		quantity          int
	}{
		{"missing user", 0, 1, 1},
		{"missing product", 1, 0, 1},
		{"zero quantity", 1, 1, 0},
		{"negative quantity", 1, 1, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 无效输入必须在获取连接、开启事务之前被拒绝。
			if err := CreateOrder(t.Context(), nil, tt.userID, tt.productID, tt.quantity); err == nil {
				t.Fatal("无效订单参数应返回错误")
			}
		})
	}
}

func TestCleanupRejectsEmpty(t *testing.T) {
	if err := cleanupDemoRecords(t.Context(), nil, demoRecords{}); err == nil {
		t.Fatal("清理缺少主键时应直接返回错误")
	}
}
