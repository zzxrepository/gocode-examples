package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"database-sql-demo/internal/store"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	baseDB, err := store.OpenMySQL(ctx, mysqlConfig())
	if err != nil {
		log.Fatal(err)
	}
	defer baseDB.Close()

	// sqlx 仅包装现有 *sql.DB，连接池仍由 database/sql 管理。
	db := store.NewSQLX(baseDB)

	product, err := store.CreateProductSQLX(ctx, db, store.CreateProductParams{
		SKU:        fmt.Sprintf("keyboard-sqlx-%d", time.Now().UnixNano()),
		Name:       "sqlx 机械键盘",
		PriceCents: 49900,
		Stock:      10,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("创建商品: id=%d stock=%d", product.ID, product.Stock)

	orderID, err := store.PlaceOrderSQLX(ctx, db, store.PlaceOrderInput{
		CustomerID: 10002,
		ProductID:  product.ID,
		Quantity:   2,
	})
	if err != nil {
		log.Fatal(err)
	}

	updated, err := store.GetProductSQLX(ctx, db, product.ID)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("下单成功: order_id=%d; 下单后库存=%d", orderID, updated.Stock)
}

func mysqlConfig() store.MySQLConfig {
	return store.MySQLConfig{
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
