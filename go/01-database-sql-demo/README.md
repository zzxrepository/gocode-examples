# Go MySQL 数据访问 Demo：database/sql

使用 Go 标准库 `database/sql`，显式演示参数绑定、结果扫描与事务。

这个独立项目使用“下单扣库存”演示数据库访问：

```text
创建商品 -> 查询与更新 -> 锁定库存 -> 扣库存 -> 创建订单 -> 写入订单项 -> 删除临时商品
```

重点 API：ExecContext、QueryRowContext、Rows、BeginTx。另一个独立示例见[配套项目](../02-sqlx-demo)。

## 文件结构

```text
01-database-sql-demo/
├── go.mod
├── main.go              # 演示入口、环境变量和执行步骤
├── mysql.go             # 驱动配置与连接池初始化
├── product.go           # 商品数据、输入校验及 CRUD
├── order.go             # 下单校验和事务
├── demo_test.go         # 不依赖 MySQL 的输入边界测试
├── docker-compose.yml   # 本示例的 MySQL 容器
├── schema.sql           # 建表脚本
└── Makefile
```

所有 Go 文件使用 `package main`，直接放在根目录。项目是命令行示例，当前不需要独立的 handler、service 或 repository 包。

## 运行

需要 Docker Compose 和 Go 1.26.5。Makefile 使用本机 Go 1.26.5，缓存放在 `.cache/go-build`。

```bash
make docker-up
# 等待 MySQL 健康检查通过后执行：
make run
# 也可以直接执行 go run .
make test
make docker-down
```

默认连接 `root:rootpass@tcp(127.0.0.1:3307)/go_store`。两个项目使用不同容器和端口，可以同时运行。首次启动容器时自动执行 `schema.sql`。

连接其他 MySQL：

```bash
MYSQL_USER=app MYSQL_PASSWORD=secret MYSQL_ADDRESS=127.0.0.1:3306 MYSQL_DATABASE=go_store make run
```

清理容器及其数据卷后重新初始化：

```bash
make docker-reset
make docker-up
```

## 下单事务

```text
BeginTx / BeginTxx
  -> SELECT ... FOR UPDATE
  -> UPDATE products SET stock = stock - quantity
  -> INSERT orders
  -> INSERT order_items
Commit

任意一步失败 -> Rollback
```

事务中的每一条 SQL 都通过 `tx` 执行，保持库存和订单的一致性。
