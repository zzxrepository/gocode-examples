package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	mysql "github.com/go-sql-driver/mysql"
)

func main() {
	initOnly := flag.Bool("init", false, "创建 person 表并补充演示数据")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := run(ctx, *initOnly); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, initOnly bool) error {
	database := os.Getenv("MYSQL_DATABASE")
	if database == "" {
		database = "go_mysql_demo"
	}
	dsn := localDSN(database)

	baseDB, err := openDatabase(ctx, dsn)
	if err != nil {
		return err
	}
	defer baseDB.Close()
	if initOnly {
		if err := initializeSchema(ctx, baseDB); err != nil {
			return err
		}
		log.Println("person 表与演示数据已就绪")
		return nil
	}
	db := baseDB
	// 后缀让两套示例都能反复运行，不依赖固定自增编号。
	suffix := time.Now().UnixNano()
	name := fmt.Sprintf("charlie_%d", suffix)
	id, err := insertPerson(ctx, db, Person{Username: name, Age: 22,
		Email: sql.NullString{String: "charlie@example.com", Valid: true}})
	if err != nil {
		return err
	}
	person, err := findPerson(ctx, db, id)
	if err != nil {
		return err
	}
	log.Printf("单行查询: id=%d username=%s age=%d", person.UserID, person.Username, person.Age)
	if _, err := updateAge(ctx, db, id, 23); err != nil {
		return err
	}
	person, err = findPerson(ctx, db, id)
	if err != nil {
		return err
	}
	log.Printf("更新后年龄: %d", person.Age)
	bob, err := findByUsername(ctx, db, "bob")
	if err != nil {
		return err
	}
	log.Printf("NULL 扫描: bob.email.Valid=%t", bob.Email.Valid)
	people, err := listPeople(ctx, db)
	if err != nil {
		return err
	}
	log.Printf("多行查询: %d 条", len(people))

	// 第二次插入相同用户名，必定触发唯一键错误；验证第一条也被回滚。
	batchName := fmt.Sprintf("batch_%d", suffix)
	err = createPeople(ctx, db, []Person{{Username: batchName, Age: 25}, {Username: batchName, Age: 26}})
	var duplicate *mysql.MySQLError
	if !errors.As(err, &duplicate) || duplicate.Number != 1062 {
		return fmt.Errorf("事务实验应返回重复键错误，实际为 %v", err)
	}
	if _, err := findByUsername(ctx, db, batchName); !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("事务回滚验证失败: %v", err)
	}
	log.Println("事务回滚: 第一条插入未保留")
	if _, err := deletePerson(ctx, db, id); err != nil {
		return err
	}
	if _, err := findPerson(ctx, db, id); !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("删除验证失败: %v", err)
	}
	log.Println("删除后查询: sql.ErrNoRows")
	return nil
}
