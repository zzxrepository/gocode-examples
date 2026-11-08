package main

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrInsufficientStock 同时表示商品不存在或库存不足。
var ErrInsufficientStock = errors.New("商品不存在或库存不足")

// CreateOrder 在同一个事务中扣减库存并创建订单。调用方应先验证用户身份。
func CreateOrder(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	productID uint,
	quantity int,
) error {
	if userID == 0 || productID == 0 {
		return errors.New("用户 ID 和商品 ID 必须有效")
	}
	if quantity <= 0 {
		return errors.New("购买数量必须大于 0")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 将库存条件与扣减放在同一条 UPDATE 中，避免先读再写。
		result := tx.
			Model(&Product{}).
			Where("id = ? AND stock >= ?", productID, quantity).
			Update("stock", gorm.Expr("stock - ?", quantity))

		if result.Error != nil {
			return fmt.Errorf("扣减库存：%w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrInsufficientStock
		}

		order := Order{
			UserID:    userID,
			ProductID: productID,
			Quantity:  quantity,
		}

		// 使用同一个 tx，让订单插入与库存扣减属于同一事务。
		if err := tx.Create(&order).Error; err != nil {
			return fmt.Errorf("创建订单：%w", err)
		}

		return nil
	})
}
