# Go MySQL 数据访问 Demo：database/sql 与 sqlx

这个项目使用“下单扣库存”演示 Go 访问 MySQL 的完整主线：

~~~text
创建商品 -> 查询与更新 -> 锁定库存 -> 扣库存 -> 创建订单 -> 写入订单项 -> 删除临时商品
~~~

它提供两个可独立运行的入口：

| 命令 | 使用的访问方式 | 关注点 |
| --- | --- | --- |
| make run | database/sql | ExecContext、QueryRowContext、Rows、BeginTx |
| make run-sqlx | sqlx | GetContext、SelectContext、命名参数、sqlx.In、BeginTxx |

两者连接同一个 MySQL，使用同一张表和同一套下单业务规则。sqlx 版本通过 sqlx.NewDb 包装已有的 *sql.DB，因此不会创建第二个连接池。

## 环境要求

- Docker 与 Docker Compose
- Go 1.26.5

Makefile 固定使用本机 Go 1.26.5，并把构建缓存放在项目内的 .cache/go-build。

## 运行

~~~bash
# 1. 启动 MySQL。schema.sql 会在首次创建数据卷时自动执行。
make docker-up

# 2. 等待健康检查通过后，运行只使用 database/sql 的版本。
make run

# 3. 运行 sqlx 版本，观察同一业务逻辑如何减少映射与绑定样板。
make run-sqlx

# 4. 运行不依赖 MySQL 的输入校验测试，并同时检查所有包可编译。
make test
~~~

make run 的输出类似：

~~~text
创建商品: id=1 stock=10
下单成功: order_id=1; 下单后库存=8
~~~

默认连接地址为 root:rootpass@tcp(127.0.0.1:3307)/go_store。也可以通过下列环境变量连接其他 MySQL：

~~~bash
MYSQL_USER=app +MYSQL_PASSWORD=secret +MYSQL_ADDRESS=127.0.0.1:3306 +MYSQL_DATABASE=go_store +make run
~~~

需要重新初始化数据时执行：

~~~bash
# 删除 Docker 数据卷；下一次 docker-up 会重新执行 schema.sql。
make docker-reset
make docker-up
~~~

## 代码入口

~~~text
cmd/database-sql-demo/main.go  标准库完整下单流程
cmd/sqlx-demo/main.go          sqlx 完整下单流程
internal/store/mysql.go        驱动配置与 database/sql 连接池
internal/store/product.go      标准库 CRUD、Rows 资源释放和结果判断
internal/store/order.go        标准库事务、FOR UPDATE 和防超卖条件
internal/store/sqlx.go         sqlx 映射、命名参数、IN 查询和事务版本
scripts/schema.sql             MySQL 表结构
~~~

PlaceOrder 和 PlaceOrderSQLX 都遵守相同的事务边界：

~~~text
BeginTx
  -> SELECT ... FOR UPDATE
  -> UPDATE products SET stock = stock - quantity
  -> INSERT orders
  -> INSERT order_items
Commit

任意一步失败 -> Rollback
~~~

事务中的每一条 SQL 都必须通过 tx 执行。若误用外层 db，该语句不会属于当前事务，库存和订单就可能失去一致性。
