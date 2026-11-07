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

	baseDB, err := OpenMySQL(ctx, mysqlConfig())
	if err != nil {
		log.Fatal(err)
	}
	defer baseDB.Close()

	// sqlx 仅包装现有 *sql.DB，连接池仍由 database/sql 管理。
	db := NewSQLX(baseDB)

	product, err := CreateProduct(ctx, db, CreateProductInput{
		SKU:        fmt.Sprintf("keyboard-sqlx-%d", time.Now().UnixNano()),
		Name:       "sqlx 机械键盘",
		PriceCents: 49900,
		Stock:      10,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("创建商品: id=%d stock=%d", product.ID, product.Stock)

	if product, err = GetProduct(ctx, db, product.ID); err != nil {
		log.Fatal(err)
	}
	log.Printf("查询商品: sku=%s name=%s", product.SKU, product.Name)

	if changed, err := UpdateProductName(ctx, db, product.ID, "sqlx 热插拔机械键盘"); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("更新商品名称: changed=%t", changed)
	}

	products, err := ListProducts(ctx, db)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("当前商品数: %d", len(products))

	orderID, err := PlaceOrder(ctx, db, PlaceOrderInput{
		CustomerID: 10002,
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

	temporary, err := CreateProduct(ctx, db, CreateProductInput{
		SKU:        fmt.Sprintf("temporary-sqlx-%d", time.Now().UnixNano()),
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

	stats := baseDB.Stats()
	log.Printf("连接池: open=%d idle=%d in_use=%d wait_count=%d", stats.OpenConnections, stats.Idle, stats.InUse, stats.WaitCount)
}

func mysqlConfig() MySQLConfig {
	return MySQLConfig{
		User:     getenv("MYSQL_USER", "root"),
		Password: getenv("MYSQL_PASSWORD", "rootpass"),
		Address:  getenv("MYSQL_ADDRESS", "127.0.0.1:3308"),
		Database: getenv("MYSQL_DATABASE", "go_store"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
