package main

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// GetUser 按主键查询；保留原始错误以便 errors.Is 判断未找到记录。
func GetUser(
	ctx context.Context,
	db *gorm.DB,
	id uint,
) (User, error) {
	var user User

	if err := db.
		WithContext(ctx).
		First(&user, id).
		Error; err != nil {
		return User{}, fmt.Errorf("查询用户 %d：%w", id, err)
	}

	return user, nil
}

// ListUsers 分别统计总数并查询当前页，两个操作从相同基础条件派生。
func ListUsers(
	ctx context.Context,
	db *gorm.DB,
	page int,
	pageSize int,
) ([]User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	base := db.
		WithContext(ctx).
		Model(&User{}).
		Where("active = ?", true).
		Session(&gorm.Session{})

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户数量：%w", err)
	}

	var users []User
	if err := base.
		Order("id DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&users).
		Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户分页：%w", err)
	}

	return users, total, nil
}
