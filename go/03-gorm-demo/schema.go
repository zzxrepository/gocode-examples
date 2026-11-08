package main

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// migrateSchema 仅用于本地教学，明确使用支持事务的 InnoDB 表。
func migrateSchema(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).
		Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(&User{}, &Product{}, &Order{}); err != nil {
		return fmt.Errorf("迁移演示表：%w", err)
	}
	return nil
}
