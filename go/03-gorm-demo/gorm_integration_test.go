package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const foreignKeyNumber uint16 = 1452

// newTestDB 每次创建独立临时库，关闭 GORM 池后再删除，不改动学习库。
func newTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	if os.Getenv("MYSQL_INTEGRATION") != "1" {
		t.Skip("设置 MYSQL_INTEGRATION=1 后连接本机 MySQL")
	}
	cfg, err := mysqldriver.ParseDSN(configuredDSN())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Net != "tcp" || cfg.Addr != "127.0.0.1:3306" {
		t.Fatal("集成测试只允许连接本机 127.0.0.1:3306")
	}
	cfg.DBName = ""
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx := t.Context()
	name := fmt.Sprintf("gorm_demo_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		if _, err := admin.ExecContext(cleanupCtx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Error(err)
		}
	})
	cfg.DBName = name
	dsn := cfg.FormatDSN()
	db, err := OpenDB(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	if pool.Stats().OpenConnections == 0 {
		t.Fatal("默认 gorm.Open 应已访问数据库")
	}
	if err := migrateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	return db, dsn
}

func TestUserOperationsMySQL(t *testing.T) {
	db, _ := newTestDB(t)
	ctx := t.Context()
	records, err := createDemoRecords(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	user := records.Users[0]
	if user.ID == 0 || user.CreatedAt.IsZero() || records.Users[1].ID == 0 {
		t.Fatal("主键或时间未回填")
	}
	if err := demoQueries(ctx, db, records.Users); err != nil {
		t.Fatal(err)
	}
	users, total, err := ListUsers(ctx, db, 0, 101)
	if err != nil || total != 2 || len(users) != 2 || users[0].ID < users[1].ID {
		t.Fatalf("分页结果 len=%d total=%d err=%v", len(users), total, err)
	}
	if err := demoUpdates(ctx, db, user.ID); err != nil {
		t.Fatal(err)
	}
	updated, err := GetUser(ctx, db, user.ID)
	if err != nil || updated.Age != 0 || updated.Active || updated.Name != "新名字" {
		t.Fatalf("更新结果不符合预期: %+v err=%v", updated, err)
	}
	if err := demoRawSQL(ctx, db, user.ID); err != nil {
		t.Fatal(err)
	}
	updated, err = GetUser(ctx, db, user.ID)
	if err != nil || !updated.Active {
		t.Fatalf("Exec 未写入 active=true: %v", err)
	}
	if err := demoSoftDelete(ctx, db, user); err != nil {
		t.Fatal(err)
	}
	if _, err := GetUser(ctx, db, user.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("软删除查询错误: %v", err)
	}
	var rawCount int64
	if err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM users WHERE id = ?", user.ID).Scan(&rawCount).Error; err != nil {
		t.Fatal(err)
	}
	if rawCount != 1 {
		t.Fatal("原生 SQL 应能看到软删除记录")
	}
	if err := cleanupDemoRecords(ctx, db, records); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Unscoped().Model(&User{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("物理清理后用户仍然存在")
	}
}

func TestCreateOrderMySQL(t *testing.T) {
	db, _ := newTestDB(t)
	ctx := t.Context()
	records, err := createDemoRecords(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	userID, productID := records.Users[0].ID, records.Product.ID
	if err := CreateOrder(ctx, db, userID, productID, 2); err != nil {
		t.Fatal(err)
	}
	var user User
	if err := db.Preload("Orders").First(&user, userID).Error; err != nil {
		t.Fatal(err)
	}
	if len(user.Orders) != 1 || user.Orders[0].Quantity != 2 {
		t.Fatal("预加载订单与下单结果不一致")
	}
	if err := CreateOrder(ctx, db, userID, productID, 4); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("库存不足错误: %v", err)
	}

	// 用已删除用户触发订单外键失败，验证之前成功执行的库存扣减也被撤销。
	missingID := records.Users[2].ID
	if err := db.Unscoped().Delete(&User{}, missingID).Error; err != nil {
		t.Fatal(err)
	}
	err = CreateOrder(ctx, db, missingID, productID, 1)
	var foreignKey *mysqldriver.MySQLError
	if !errors.As(err, &foreignKey) || foreignKey.Number != foreignKeyNumber {
		t.Fatalf("应为订单外键错误，实际为 %v", err)
	}
	var product Product
	if err := db.First(&product, productID).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&Order{}).Where("product_id = ?", productID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 3 || count != 1 {
		t.Fatalf("失败写入未正确回滚: stock=%d orders=%d", product.Stock, count)
	}
}

func TestCreateOrderDatabaseError(t *testing.T) {
	db, _ := newTestDB(t)
	records, err := createDemoRecords(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	// 在隔离测试库中模拟业务表不可用，确认事务中的 UPDATE 错误向上传播。
	if err := db.Migrator().DropTable(&Product{}); err != nil {
		t.Fatal(err)
	}
	err = CreateOrder(t.Context(), db, records.Users[0].ID, records.Product.ID, 1)
	var missingTable *mysqldriver.MySQLError
	const missingTableNumber uint16 = 1146
	if !errors.As(err, &missingTable) || missingTable.Number != missingTableNumber {
		t.Fatalf("未保留库存更新的数据库错误: %v", err)
	}
}

func TestCreateOrderConcurrent(t *testing.T) {
	db, _ := newTestDB(t)
	records, err := createDemoRecords(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	const requests = 8
	results := make(chan error, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Go(func() { results <- CreateOrder(t.Context(), db, records.Users[0].ID, records.Product.ID, 1) })
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrInsufficientStock):
		default:
			t.Fatal(err)
		}
	}
	var product Product
	if err := db.First(&product, records.Product.ID).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&Order{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if succeeded != records.Product.Stock || product.Stock != 0 || count != int64(succeeded) {
		t.Fatalf("并发库存不一致: succeeded=%d stock=%d orders=%d", succeeded, product.Stock, count)
	}
}

func TestRunMySQL(t *testing.T) {
	db, dsn := newTestDB(t)
	t.Setenv("MYSQL_DSN", dsn)
	ctx := t.Context()
	existing := User{Name: "已有用户", Email: "existing@example.com", Age: 30, Active: true}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	product := Product{Name: "已有商品", Stock: 99}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	for _, opts := range []struct{ initOnly, showSQL bool }{{true, false}, {false, false}, {false, true}} {
		if err := run(ctx, opts.initOnly, opts.showSQL); err != nil {
			t.Fatal(err)
		}
	}
	got, err := GetUser(ctx, db, existing.ID)
	if err != nil || got.Name != existing.Name {
		t.Fatal("演示影响了已有用户")
	}
	var count int64
	if err := db.Unscoped().Model(&User{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("演示未清理干净，users=%d", count)
	}
	if err := db.Model(&Product{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("演示影响了已有商品或未清理，products=%d", count)
	}
	if err := db.First(&product, product.ID).Error; err != nil {
		t.Fatal(err)
	}
	if product.Stock != 99 {
		t.Fatal("演示修改了已有商品库存")
	}
}

func TestCanceledOperationsMySQL(t *testing.T) {
	db, _ := newTestDB(t)
	records, err := createDemoRecords(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	id := records.Users[0].ID
	cases := []struct {
		name string
		call func() error
	}{
		{"GetUser", func() error { _, err := GetUser(ctx, db, id); return err }},
		{"ListUsers", func() error { _, _, err := ListUsers(ctx, db, 1, 20); return err }},
		{"CreateOrder", func() error { return CreateOrder(ctx, db, id, records.Product.ID, 1) }},
		{"runDemo", func() error { return runDemo(ctx, db) }},
		{"migrateSchema", func() error { return migrateSchema(ctx, db) }},
		{"demoUpdates", func() error { return demoUpdates(ctx, db, id) }},
		{"demoRawSQL", func() error { return demoRawSQL(ctx, db, id) }},
		{"demoSoftDelete", func() error { return demoSoftDelete(ctx, db, records.Users[0]) }},
		{"demoAssociations", func() error { return demoAssociations(ctx, db, id) }},
		{"cleanup", func() error { return cleanupDemoRecords(ctx, db, records) }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, context.Canceled) {
				t.Fatalf("取消错误未保留: %v", err)
			}
		})
	}
}
