package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

func demoTransactions(ctx context.Context, db *gorm.DB, userID, productID uint) error {
	if err := CreateOrder(ctx, db, userID, productID, 2); err != nil {
		return err
	}
	var product Product
	if err := db.WithContext(ctx).First(&product, productID).Error; err != nil {
		return err
	}
	log.Printf("事务提交: 创建订单成功，剩余库存=%d", product.Stock)
	err := CreateOrder(ctx, db, userID, productID, 10)
	if !errors.Is(err, ErrInsufficientStock) {
		return fmt.Errorf("库存不足时应拒绝下单，实际为 %v", err)
	}
	if err := db.WithContext(ctx).First(&product, productID).Error; err != nil {
		return err
	}
	log.Printf("库存不足: 拒绝下单，剩余库存仍为 %d", product.Stock)
	return nil
}

func demoRollback(ctx context.Context, db *gorm.DB) error {
	user := User{Name: "回滚用户", Email: fmt.Sprintf("rollback_%d@example.com", time.Now().UnixNano())}
	rejected := errors.New("演示业务校验失败")
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		// 已执行 INSERT 后返回业务错误，观察该次写入是否被回滚。
		return rejected
	})
	if !errors.Is(err, rejected) {
		return fmt.Errorf("事务应返回演示业务错误，实际为 %v", err)
	}
	var count int64
	if err := db.WithContext(ctx).Unscoped().Model(&User{}).
		Where("id = ?", user.ID).Count(&count).Error; err != nil {
		return err
	}
	if count != 0 {
		return errors.New("回滚验证失败：插入记录仍然存在")
	}
	log.Println("事务回滚: INSERT 后返回错误，该记录未保留")
	return nil
}

func demoAssociations(ctx context.Context, db *gorm.DB, id uint) error {
	var user User
	if err := db.WithContext(ctx).Preload("Orders").First(&user, id).Error; err != nil {
		return err
	}
	log.Printf("Preload: 当前用户关联订单=%d 条", len(user.Orders))
	return nil
}
