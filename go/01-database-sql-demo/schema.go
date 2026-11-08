package main

import (
	"context"
	"database/sql"
	"fmt"
)

// 只在显式执行 -init 时初始化 person 表。
const createPersonTableSQL = `CREATE TABLE IF NOT EXISTS person (
    user_id BIGINT NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    age INT NOT NULL,
    email VARCHAR(128) NULL,
    PRIMARY KEY (user_id),
    UNIQUE KEY uk_person_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

func initializeSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createPersonTableSQL); err != nil {
		return fmt.Errorf("create person: %w", err)
	}
	// 固定、合法的演示数据。重复初始化时跳过同名记录，不重置已有字段。
	_, err := db.ExecContext(ctx, `INSERT IGNORE INTO person (username, age, email)
        VALUES ('alice', 20, 'alice@example.com'), ('bob', 21, NULL)`)
	if err != nil {
		return fmt.Errorf("seed person: %w", err)
	}
	return nil
}
