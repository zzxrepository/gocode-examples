package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidProduct  = errors.New("invalid product")
)

// Product 同时服务 database/sql 和 sqlx。db tag 会在 sqlx 映射时使用。
type Product struct {
	ID         int64     `db:"id"`
	SKU        string    `db:"sku"`
	Name       string    `db:"name"`
	PriceCents int64     `db:"price_cents"`
	Stock      int64     `db:"stock"`
	CreatedAt  time.Time `db:"created_at"`
}

type CreateProductInput struct {
	SKU        string
	Name       string
	PriceCents int64
	Stock      int64
}

func validateProductInput(input CreateProductInput) error {
	if input.SKU == "" || input.Name == "" || input.PriceCents < 0 || input.Stock < 0 {
		return ErrInvalidProduct
	}
	return nil
}

// CreateProduct 展示 database/sql 的 ExecContext 和 LastInsertId。
func CreateProduct(ctx context.Context, db *sql.DB, input CreateProductInput) (Product, error) {
	if err := validateProductInput(input); err != nil {
		return Product{}, err
	}

	result, err := db.ExecContext(ctx, `
		INSERT INTO products (sku, name, price_cents, stock)
		VALUES (?, ?, ?, ?)`,
		input.SKU, input.Name, input.PriceCents, input.Stock,
	)
	if err != nil {
		return Product{}, fmt.Errorf("insert product: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Product{}, fmt.Errorf("read inserted product ID: %w", err)
	}
	return GetProduct(ctx, db, id)
}

// GetProduct 展示 QueryRowContext：查询错误和 sql.ErrNoRows 都在 Scan 时出现。
func GetProduct(ctx context.Context, db *sql.DB, id int64) (Product, error) {
	var product Product
	err := db.QueryRowContext(ctx, `
		SELECT id, sku, name, price_cents, stock, created_at
		FROM products
		WHERE id = ?`, id,
	).Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.PriceCents,
		&product.Stock,
		&product.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product %d: %w", id, err)
	}
	return product, nil
}

// ListProducts 展示 QueryContext、Rows.Close 和 Rows.Err 的完整模式。
func ListProducts(ctx context.Context, db *sql.DB) ([]Product, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, sku, name, price_cents, stock, created_at
		FROM products
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query products: %w", err)
	}
	defer rows.Close() // 释放结果集，连接才能尽快回到连接池。

	products := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(
			&product.ID,
			&product.SKU,
			&product.Name,
			&product.PriceCents,
			&product.Stock,
			&product.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan product row: %w", err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product rows: %w", err)
	}
	return products, nil
}
