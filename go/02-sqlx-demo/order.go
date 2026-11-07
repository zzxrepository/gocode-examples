package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/jmoiron/sqlx"
)

var (
	ErrInvalidOrder      = errors.New("invalid order")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type PlaceOrderInput struct {
	CustomerID int64
	ProductID  int64
	Quantity   int64
}

func validateOrderInput(input PlaceOrderInput) error {
	if input.CustomerID <= 0 || input.ProductID <= 0 || input.Quantity <= 0 {
		return ErrInvalidOrder
	}
	return nil
}

// PlaceOrder 使用 sqlx 事务完成锁库存、扣库存、创建订单和订单明细。
func PlaceOrder(ctx context.Context, db *sqlx.DB, input PlaceOrderInput) (int64, error) {
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
