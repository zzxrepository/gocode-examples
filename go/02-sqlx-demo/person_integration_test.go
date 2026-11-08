package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// MySQL 驱动的 OpenConnector 只解析 DSN。即使目标端口不可用，Open 也不拨号。
func TestOpenDoesNotDial(t *testing.T) {
	db, err := sql.Open("mysql", "root:unused@tcp(127.0.0.1:1)/unused")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if got := db.Stats().OpenConnections; got != 0 {
		t.Fatalf("OpenConnections=%d, want 0 before Ping/query", got)
	}
}

// MYSQL_INTEGRATION=1 时使用本机账号运行；每次创建并删除独立测试库。
func TestPersonMySQL(t *testing.T) {
	if os.Getenv("MYSQL_INTEGRATION") != "1" {
		t.Skip("set MYSQL_INTEGRATION=1 to run local MySQL integration checks")
	}
	cfg, err := mysql.ParseDSN(localDSN(""))
	if err != nil {
		t.Fatal(err)
	}
	cfg.DBName = ""
	cfg.ClientFoundRows = false
	admin, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	name := fmt.Sprintf("go_person_demo_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := admin.ExecContext(cleanupCtx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Errorf("drop isolated test database: %v", err)
		}
	}()
	cfg.DBName = name
	baseDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer baseDB.Close()
	// 结果集泄漏或事务内误用外层 db，会在单连接池中暴露为超时。
	baseDB.SetMaxOpenConns(1)
	baseDB.SetMaxIdleConns(1)
	if err := initializeSchema(ctx, baseDB); err != nil {
		t.Fatal(err)
	}
	if err := initializeSchema(ctx, baseDB); err != nil {
		t.Fatalf("repeat initialization: %v", err)
	}
	db := sqlx.NewDb(baseDB, "mysql")
	bob, err := findByUsername(ctx, db, "bob")
	if err != nil {
		t.Fatal(err)
	}
	if bob.Email.Valid {
		t.Fatalf("bob email=%+v, want SQL NULL", bob.Email)
	}
	id, err := insertPerson(ctx, db, Person{Username: "charlie", Age: 22, Email: sql.NullString{String: "", Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := findPerson(ctx, db, id)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Email.Valid || got.Email.String != "" {
		t.Fatalf("empty string conflated with NULL: %+v", got.Email)
	}
	if n, err := updateAge(ctx, db, id, 23); err != nil || n != 1 {
		t.Fatalf("update=%d,%v", n, err)
	}
	if n, err := updateAge(ctx, db, id, 23); err != nil || n != 0 {
		t.Fatalf("same-value update=%d,%v", n, err)
	}
	// 显式占住唯一连接，验证后续查询会在池中等待并响应 Context 超时。
	held, err := baseDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	waitCtx, stopWait := context.WithTimeout(ctx, 20*time.Millisecond)
	var one int
	waitErr := baseDB.QueryRowContext(waitCtx, "SELECT 1").Scan(&one)
	stopWait()
	if err := held.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Fatalf("pool wait error=%v", waitErr)
	}
	if baseDB.Stats().WaitCount == 0 {
		t.Fatal("pool wait counter did not increase")
	}
	people, err := listPeople(ctx, db)
	if err != nil || len(people) != 3 {
		t.Fatalf("list len=%d err=%v", len(people), err)
	}
	selected, err := findPeopleByIDs(ctx, db, []int64{id, bob.UserID})
	if err != nil || len(selected) != 2 {
		t.Fatalf("IN len=%d err=%v", len(selected), err)
	}
	visited := 0
	if err := walkPeople(ctx, db, func(Person) error { visited++; return nil }); err != nil {
		t.Fatal(err)
	}
	if visited != 3 {
		t.Fatalf("walk count=%d", visited)
	}
	callbackErr := errors.New("stop after first row")
	if err := walkPeople(ctx, db, func(Person) error { return callbackErr }); !errors.Is(err, callbackErr) {
		t.Fatalf("callback err=%v", err)
	}
	// 上一次回调提前退出后，单连接池仍应能立即继续查询。
	if _, err := findPerson(ctx, db, id); err != nil {
		t.Fatalf("rows not released after callback error: %v", err)
	}

	if err := createPeople(ctx, db, []Person{{Username: "ok_first", Age: 30}, {Username: "ok_second", Age: 31}}); err != nil {
		t.Fatal(err)
	}
	if _, err := findByUsername(ctx, db, "ok_second"); err != nil {
		t.Fatalf("commit not visible: %v", err)
	}
	err = createPeople(ctx, db, []Person{{Username: "rolled_back", Age: 25}, {Username: "rolled_back", Age: 26}})
	var duplicate *mysql.MySQLError
	if !errors.As(err, &duplicate) || duplicate.Number != 1062 {
		t.Fatalf("want duplicate key error, got %v", err)
	}
	if _, err := findByUsername(ctx, db, "rolled_back"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("partial transaction remained: %v", err)
	}
	if n, err := deletePerson(ctx, db, id); err != nil || n != 1 {
		t.Fatalf("delete=%d,%v", n, err)
	}
	if _, err := findPerson(ctx, db, id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("want wrapped ErrNoRows, got %v", err)
	}
	t.Setenv("MYSQL_USER", cfg.User)
	t.Setenv("MYSQL_PASSWORD", cfg.Passwd)
	t.Setenv("MYSQL_DATABASE", name)
	if err := run(ctx, false); err != nil {
		t.Fatalf("full demo run: %v", err)
	}

}

func TestEmptyIDs(t *testing.T) {
	got, err := findPeopleByIDs(context.Background(), nil, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty ids: %v, %v", got, err)
	}
}
