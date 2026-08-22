package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/jmoiron/sqlx"
)

// NewSQLX 只包装现有 *sql.DB，因此两者共享同一连接池。
func NewSQLX(db *sql.DB) *sqlx.DB {
	return sqlx.NewDb(db, "mysql")
}

type CreateProductParams struct {
	SKU        string `db:"sku"`
	Name       string `db:"name"`
	PriceCents int64  `db:"price_cents"`
	Stock      int64  `db:"stock"`
}

// CreateProductSQLX 用命名参数替换长位置参数，SQL 本身依旧显式存在。
func CreateProductSQLX(ctx context.Context, db *sqlx.DB, input CreateProductParams) (Product, error) {
	if err := validateProductInput(CreateProductInput(input)); err != nil {
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
	return GetProductSQLX(ctx, db, id)
}

// GetProductSQLX 用 db tag 将查询列映射到 Product，不再手写 Scan 顺序。
func GetProductSQLX(ctx context.Context, db *sqlx.DB, id int64) (Product, error) {
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

func ListProductsSQLX(ctx context.Context, db *sqlx.DB) ([]Product, error) {
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

// UpdateProductNameSQLX 与标准库版本语义相同，但名称和 ID 改由命名参数绑定。
func UpdateProductNameSQLX(ctx context.Context, db *sqlx.DB, id int64, name string) (bool, error) {
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

// DeleteProductSQLX 展示 sqlx 仍可直接执行普通 SQL；它不会替应用生成 SQL。
func DeleteProductSQLX(ctx context.Context, db *sqlx.DB, id int64) (bool, error) {
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

func GetProductsByIDsSQLX(ctx context.Context, db *sqlx.DB, ids []int64) ([]Product, error) {
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

// PlaceOrderSQLX 与 PlaceOrder 的业务规则相同，只替换结构体映射和命名参数。
func PlaceOrderSQLX(ctx context.Context, db *sqlx.DB, input PlaceOrderInput) (int64, error) {
	if err := validateOrderInput(input); err != nil {
		return 0, err
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin order transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var locked struct {
		PriceCents int64 `db:"price_cents"`
		Stock      int64 `db:"stock"`
	}
	if err := tx.GetContext(ctx, &locked, `
		SELECT price_cents, stock
		FROM products
		WHERE id = ?
		FOR UPDATE`, input.ProductID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrProductNotFound
		}
		return 0, fmt.Errorf("lock product %d: %w", input.ProductID, err)
	}
	if locked.Stock < input.Quantity {
		return 0, ErrInsufficientStock
	}
	if locked.PriceCents > math.MaxInt64/input.Quantity {
		return 0, ErrInvalidOrder
	}

	result, err := tx.NamedExecContext(ctx, `
		UPDATE products
		SET stock = stock - :quantity
		WHERE id = :product_id AND stock >= :quantity`, map[string]any{
		"product_id": input.ProductID,
		"quantity":   input.Quantity,
	})
	if err != nil {
		return 0, fmt.Errorf("decrease product stock: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read stock update result: %w", err)
	}
	if affected != 1 {
		return 0, ErrInsufficientStock
	}

	total := locked.PriceCents * input.Quantity
	orderResult, err := tx.NamedExecContext(ctx, `
		INSERT INTO orders (customer_id, status, total_amount_cents)
		VALUES (:customer_id, :status, :total)`, map[string]any{
		"customer_id": input.CustomerID,
		"status":      "created",
		"total":       total,
	})
	if err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}
	orderID, err := orderResult.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted order ID: %w", err)
	}
	if _, err := tx.NamedExecContext(ctx, `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents)
		VALUES (:order_id, :product_id, :quantity, :unit_price_cents)`, map[string]any{
		"order_id":         orderID,
		"product_id":       input.ProductID,
		"quantity":         input.Quantity,
		"unit_price_cents": locked.PriceCents,
	}); err != nil {
		return 0, fmt.Errorf("insert order item: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit order transaction: %w", err)
	}
	return orderID, nil
}
