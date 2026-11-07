package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := OpenMySQL(ctx, mysqlConfig())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close() // 进程退出时关闭整个连接池，而不是每次查询后关闭。

	product, err := CreateProduct(ctx, db, CreateProductInput{
		SKU:        fmt.Sprintf("keyboard-sql-%d", time.Now().UnixNano()),
		Name:       "database/sql 机械键盘",
		PriceCents: 39900,
		Stock:      10,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("创建商品: id=%d stock=%d", product.ID, product.Stock)

	// QueryRowContext 的读取逻辑封装在 GetProduct 中；先读取一次确认写入结果。
	if product, err = GetProduct(ctx, db, product.ID); err != nil {
		log.Fatal(err)
	}
	log.Printf("查询商品: sku=%s name=%s", product.SKU, product.Name)

	if changed, err := UpdateProductName(ctx, db, product.ID, "database/sql 热插拔机械键盘"); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("更新商品名称: changed=%t", changed)
	}

	// QueryContext 的完整 Rows 迭代和关闭逻辑封装在 ListProducts 中。
	products, err := ListProducts(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("当前商品数: %d", len(products))

	orderID, err := PlaceOrder(ctx, db, PlaceOrderInput{
		CustomerID: 10001,
		ProductID:  product.ID,
		Quantity:   2,
	})
	if err != nil {
		log.Fatal(err)
	}

	updated, err := GetProduct(ctx, db, product.ID)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("下单成功: order_id=%d; 下单后库存=%d", orderID, updated.Stock)

	// 用一件从未下单的临时商品演示 DELETE，避免外键约束删除订单历史。
	temporary, err := CreateProduct(ctx, db, CreateProductInput{
		SKU:        fmt.Sprintf("temporary-sql-%d", time.Now().UnixNano()),
		Name:       "待删除演示商品",
		PriceCents: 1,
		Stock:      0,
	})
	if err != nil {
		log.Fatal(err)
	}
	if deleted, err := DeleteProduct(ctx, db, temporary.ID); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("删除临时商品: deleted=%t", deleted)
	}

	stats := db.Stats()
	log.Printf("连接池: open=%d idle=%d in_use=%d wait_count=%d", stats.OpenConnections, stats.Idle, stats.InUse, stats.WaitCount)
}

func mysqlConfig() MySQLConfig {
	return MySQLConfig{
		User:     getenv("MYSQL_USER", "root"),
		Password: getenv("MYSQL_PASSWORD", "rootpass"),
		Address:  getenv("MYSQL_ADDRESS", "127.0.0.1:3307"),
		Database: getenv("MYSQL_DATABASE", "go_store"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
