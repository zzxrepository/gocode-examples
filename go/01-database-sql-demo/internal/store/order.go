package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
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

// PlaceOrder 原子地锁库存、扣库存、创建订单和订单明细。
func PlaceOrder(ctx context.Context, db *sql.DB, input PlaceOrderInput) (int64, error) {
	if err := validateOrderInput(input); err != nil {
		return 0, err
	}

	// 事务在 Commit 或 Rollback 前独占连接池中的一条连接。
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin order transaction: %w", err)
	}
	// Commit 成功后的 Rollback 会返回 sql.ErrTxDone；忽略即可。
	// 这个 defer 确保任意中途 return 都会撤销已经执行的写入。
	defer func() { _ = tx.Rollback() }()

	var priceCents, stock int64
	err = tx.QueryRowContext(ctx, `
		SELECT price_cents, stock
		FROM products
		WHERE id = ?
		FOR UPDATE`, input.ProductID,
	).Scan(&priceCents, &stock)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrProductNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock product %d: %w", input.ProductID, err)
	}
	if stock < input.Quantity {
		return 0, ErrInsufficientStock
	}
	if priceCents > math.MaxInt64/input.Quantity {
		return 0, ErrInvalidOrder
	}

	// 每条需要原子性的 SQL 都使用 tx，绝不能误用外层 db。
	// stock >= ? 是对库存条件的第二层保护，RowsAffected 应当恰好为 1。
	result, err := tx.ExecContext(ctx, `
		UPDATE products
		SET stock = stock - ?
		WHERE id = ? AND stock >= ?`,
		input.Quantity, input.ProductID, input.Quantity,
	)
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

	total := priceCents * input.Quantity
	orderResult, err := tx.ExecContext(ctx, `
		INSERT INTO orders (customer_id, status, total_amount_cents)
		VALUES (?, ?, ?)`, input.CustomerID, "created", total)
	if err != nil {
		return 0, fmt.Errorf("insert order: %w", err)
	}
	orderID, err := orderResult.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted order ID: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO order_items (order_id, product_id, quantity, unit_price_cents)
		VALUES (?, ?, ?, ?)`,
		orderID, input.ProductID, input.Quantity, priceCents,
	); err != nil {
		return 0, fmt.Errorf("insert order item: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit order transaction: %w", err)
	}
	return orderID, nil
}
