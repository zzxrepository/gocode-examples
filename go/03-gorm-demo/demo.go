package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

const cleanupTimeout = 5 * time.Second

// demoRecords 只记录本次创建的数据，清理时不会删除其他实验记录。
type demoRecords struct {
	Users   []User
	Product Product
}

func runDemo(ctx context.Context, db *gorm.DB) (err error) {
	records, err := createDemoRecords(ctx, db)
	if err != nil {
		return err
	}
	// 即使演示超时，也给清理单独的短超时；仅按本次生成的主键清理。
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer cancel()
		err = errors.Join(err, cleanupDemoRecords(cleanupCtx, db, records))
	}()
	if err := demoQueries(ctx, db, records.Users); err != nil {
		return err
	}
	user := records.Users[0]
	if err := demoUpdates(ctx, db, user.ID); err != nil {
		return err
	}
	if err := demoTransactions(ctx, db, user.ID, records.Product.ID); err != nil {
		return err
	}
	if err := demoRollback(ctx, db); err != nil {
		return err
	}
	if err := demoAssociations(ctx, db, user.ID); err != nil {
		return err
	}
	if err := demoRawSQL(ctx, db, user.ID); err != nil {
		return err
	}
	return demoSoftDelete(ctx, db, user)
}

func createDemoRecords(ctx context.Context, db *gorm.DB) (demoRecords, error) {
	suffix := time.Now().UnixNano()
	user := User{Name: "张三", Email: fmt.Sprintf("zhangsan_%d@example.com", suffix), Age: 20, Active: true}
	others := []User{
		{Name: "李四", Email: fmt.Sprintf("lisi_%d@example.com", suffix), Age: 22, Active: true},
		{Name: "王五", Email: fmt.Sprintf("wangwu_%d@example.com", suffix), Age: 0, Active: false},
	}
	product := Product{Name: "演示商品", Stock: 5}
	// 把演示数据的准备也放入事务，避免初始化一半就失败。
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return fmt.Errorf("创建用户：%w", err)
		}
		if err := tx.Create(&others).Error; err != nil {
			return fmt.Errorf("批量创建用户：%w", err)
		}
		if err := tx.Create(&product).Error; err != nil {
			return fmt.Errorf("创建商品：%w", err)
		}
		return nil
	})
	if err != nil {
		return demoRecords{}, err
	}
	log.Printf("Create: 主键已回填=%t；批量创建=%d 条", user.ID != 0, len(others))
	return demoRecords{Users: append([]User{user}, others...), Product: product}, nil
}

func cleanupDemoRecords(ctx context.Context, db *gorm.DB, records demoRecords) error {
	ids := userIDs(records.Users)
	if len(ids) == 0 || records.Product.ID == 0 {
		return errors.New("演示数据清理缺少主键")
	}
	// 先删除子表订单，再物理删除用户，满足用户外键约束。
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id IN ?", ids).Delete(&Order{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&User{}, ids).Error; err != nil {
			return err
		}
		return tx.Delete(&Product{}, records.Product.ID).Error
	})
	if err != nil {
		return fmt.Errorf("清理本次演示数据：%w", err)
	}
	log.Println("清理完成: 仅删除本次演示创建的用户、订单和商品")
	return nil
}

func userIDs(users []User) []uint {
	ids := make([]uint, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}
