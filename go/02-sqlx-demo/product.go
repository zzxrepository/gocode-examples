package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrInvalidProduct  = errors.New("invalid product")
)

// Product 的 db tag 用于 sqlx 将查询结果映射到结构体。
type Product struct {
	ID         int64     `db:"id"`
	SKU        string    `db:"sku"`
	Name       string    `db:"name"`
	PriceCents int64     `db:"price_cents"`
	Stock      int64     `db:"stock"`
	CreatedAt  time.Time `db:"created_at"`
}

type CreateProductInput struct {
	SKU        string `db:"sku"`
	Name       string `db:"name"`
	PriceCents int64  `db:"price_cents"`
	Stock      int64  `db:"stock"`
}

func validateProductInput(input CreateProductInput) error {
	if input.SKU == "" || input.Name == "" || input.PriceCents < 0 || input.Stock < 0 {
		return ErrInvalidProduct
	}
	return nil
}

// CreateProduct 用命名参数替换长位置参数，SQL 本身依旧显式存在。
func CreateProduct(ctx context.Context, db *sqlx.DB, input CreateProductInput) (Product, error) {
	if err := validateProductInput(input); err != nil {
		return Product{}, err
	}
	result, err := db.NamedExecContext(ctx, `
		INSERT INTO products (sku, name, price_cents, stock)
		VALUES (:sku, :name, :price_cents, :stock)`, input)
	if err != nil {
		return Product{}, fmt.Errorf("insert product: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Product{}, fmt.Errorf("read inserted product ID: %w", err)
	}
	return GetProduct(ctx, db, id)
}

// GetProduct 用 db tag 将查询列映射到 Product，不再手写 Scan 顺序。
func GetProduct(ctx context.Context, db *sqlx.DB, id int64) (Product, error) {
	var product Product
	err := db.GetContext(ctx, &product, `
		SELECT id, sku, name, price_cents, stock, created_at
		FROM products
		WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrProductNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("get product %d: %w", id, err)
	}
	return product, nil
}

func ListProducts(ctx context.Context, db *sqlx.DB) ([]Product, error) {
	products := make([]Product, 0)
	// SelectContext 内部迭代 Rows、执行 StructScan 并关闭结果集。
	if err := db.SelectContext(ctx, &products, `
		SELECT id, sku, name, price_cents, stock, created_at
		FROM products
		ORDER BY id`); err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	return products, nil
}

// UpdateProductName 使用命名参数更新商品名称。
func UpdateProductName(ctx context.Context, db *sqlx.DB, id int64, name string) (bool, error) {
	if id <= 0 || name == "" {
		return false, ErrInvalidProduct
	}

	result, err := db.NamedExecContext(ctx, `
		UPDATE products
		SET name = :name
		WHERE id = :id`, map[string]any{
		"id":   id,
		"name": name,
	})
	if err != nil {
		return false, fmt.Errorf("update product %d: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read product update result: %w", err)
	}
	return affected > 0, nil
}

// DeleteProduct 展示 sqlx 仍可直接执行普通 SQL；它不会替应用生成 SQL。
func DeleteProduct(ctx context.Context, db *sqlx.DB, id int64) (bool, error) {
	if id <= 0 {
		return false, ErrInvalidProduct
	}

	result, err := db.ExecContext(ctx, `DELETE FROM products WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete product %d: %w", id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read product delete result: %w", err)
	}
	return affected == 1, nil
}

func GetProductsByIDs(ctx context.Context, db *sqlx.DB, ids []int64) ([]Product, error) {
	if len(ids) == 0 {
		return []Product{}, nil // 防止拼出 MySQL 不接受的 IN ()。
	}
	query, args, err := sqlx.In(`
		SELECT id, sku, name, price_cents, stock, created_at
		FROM products
		WHERE id IN (?)
		ORDER BY id`, ids)
	if err != nil {
		return nil, fmt.Errorf("expand IN: %w", err)
	}
	query = db.Rebind(query) // MySQL 保持 ?；其他驱动可能改写为 $1、$2……

	products := make([]Product, 0)
	if err := db.SelectContext(ctx, &products, query, args...); err != nil {
		return nil, fmt.Errorf("query products by IDs: %w", err)
	}
	return products, nil
}
