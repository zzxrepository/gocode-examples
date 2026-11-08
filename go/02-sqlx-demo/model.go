package main

import "database/sql"

// Person 对应 person 表。db 标签由 sqlx 读取，标准库手动 Scan 不读取标签。
// Email 同时保存字符串与有效标记，从而区分 SQL NULL 和空字符串。
type Person struct {
	UserID   int64          `db:"user_id"`
	Username string         `db:"username"`
	Age      int            `db:"age"`
	Email    sql.NullString `db:"email"`
}
