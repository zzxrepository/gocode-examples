package main

import (
	"time"

	"gorm.io/gorm"
)

// User 声明软删除和一对多订单关系；Orders 不是 users 表的一列。
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:64;not null"`
	Email     string `gorm:"size:128;not null;uniqueIndex"`
	Age       int    `gorm:"not null"`
	Active    bool   `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Orders    []Order        `gorm:"foreignKey:UserID"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Product struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"size:128;not null"`
	Stock int    `gorm:"not null"`
}

type Order struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null;index"`
	ProductID uint `gorm:"not null;index"`
	Quantity  int  `gorm:"not null"`
	CreatedAt time.Time
}

type UserSummary struct {
	Name string
	Age  int
}
