package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const duplicateKeyNumber uint16 = 1062

func demoQueries(ctx context.Context, db *gorm.DB, users []User) error {
	user, err := GetUser(ctx, db, users[0].ID)
	if err != nil {
		return err
	}
	log.Printf("First: name=%s age=%d", user.Name, user.Age)
	page, total, err := ListUsers(ctx, db, 1, 20)
	if err != nil {
		return err
	}
	log.Printf("Find + Count 分页: 当前页=%d 总数=%d", len(page), total)

	// 限定本次演示的用户，使零值条件的对比不受已有数据影响。
	base := db.WithContext(ctx).Where("id IN ?", userIDs(users)).Session(&gorm.Session{})
	var byStruct, byMap []User
	if err := base.Where(&User{Age: 0, Active: false}).Find(&byStruct).Error; err != nil {
		return err
	}
	if err := base.Where(map[string]any{"age": 0, "active": false}).Find(&byMap).Error; err != nil {
		return err
	}
	log.Printf("零值查询: struct=%d 条 map=%d 条", len(byStruct), len(byMap))

	var adults, minors []User
	if err := base.Where("age >= ?", 18).Find(&adults).Error; err != nil {
		return err
	}
	if err := base.Where("age < ?", 18).Find(&minors).Error; err != nil {
		return err
	}
	log.Printf("Session 条件隔离: 成年=%d 未成年=%d", len(adults), len(minors))
	return nil
}

func demoUpdates(ctx context.Context, db *gorm.DB, id uint) error {
	result := db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Update("name", "单字段更新")
	if result.Error != nil {
		return result.Error
	}
	log.Printf("Update: 单字段更新影响行数=%d", result.RowsAffected)

	// 结构体更新会忽略 Age=0、Active=false。
	if err := db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Updates(User{Name: "新名字", Age: 0, Active: false}).Error; err != nil {
		return err
	}
	user, err := GetUser(ctx, db, id)
	if err != nil {
		return err
	}
	log.Printf("结构体更新后: age=%d active=%t", user.Age, user.Active)
	if err := db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Updates(map[string]any{"age": 0, "active": false}).Error; err != nil {
		return err
	}
	user, err = GetUser(ctx, db, id)
	if err != nil {
		return err
	}
	log.Printf("Map 更新后: age=%d active=%t", user.Age, user.Active)

	// 先恢复非零值，再展示 Select 如何要求写入结构体的零值。
	if err := db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Updates(map[string]any{"age": 20, "active": true}).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Model(&User{}).Where("id = ?", id).
		Select("Age", "Active").Updates(User{Age: 0, Active: false}).Error; err != nil {
		return err
	}
	user, err = GetUser(ctx, db, id)
	if err != nil {
		return err
	}
	log.Printf("Select 更新后: age=%d active=%t", user.Age, user.Active)
	return nil
}

func demoRawSQL(ctx context.Context, db *gorm.DB, id uint) error {
	var summaries []UserSummary
	// 原生 SQL 自己声明软删除条件；Scan 才会触发查询。
	if err := db.WithContext(ctx).Raw(
		"SELECT name, age FROM users WHERE age >= ? AND deleted_at IS NULL", 18,
	).Scan(&summaries).Error; err != nil {
		return err
	}
	result := db.WithContext(ctx).Exec(
		"UPDATE users SET active = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
		true, time.Now(), id,
	)
	if result.Error != nil {
		return result.Error
	}
	log.Printf("Raw + Scan: %d 条；Exec 影响行数=%d", len(summaries), result.RowsAffected)
	return nil
}

func demoSoftDelete(ctx context.Context, db *gorm.DB, user User) error {
	if err := db.WithContext(ctx).Delete(&User{}, user.ID).Error; err != nil {
		return err
	}
	if _, err := GetUser(ctx, db, user.ID); !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("软删除后应返回 ErrRecordNotFound，实际为 %v", err)
	}
	var deleted User
	if err := db.WithContext(ctx).Unscoped().First(&deleted, user.ID).Error; err != nil {
		return err
	}
	log.Printf("软删除: 普通查询未找到；Unscoped 的 DeletedAt.Valid=%t", deleted.DeletedAt.Valid)

	// 原记录尚在，邮箱唯一索引仍会阻止重复使用。
	sameEmail := User{Name: "重复邮箱", Email: user.Email}
	err := db.WithContext(ctx).Create(&sameEmail).Error
	var duplicate *mysqldriver.MySQLError
	if !errors.As(err, &duplicate) || duplicate.Number != duplicateKeyNumber {
		return fmt.Errorf("软删除后应保留邮箱唯一约束，实际为 %v", err)
	}
	log.Println("软删除后复用邮箱: MySQL 1062，唯一值仍被原记录占用")
	return nil
}
