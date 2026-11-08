# Go 操作 MySQL：sqlx

使用单张 `person` 表演示连接、增删改查、SQL NULL 与空字符串、缺失记录和事务提交/回滚。

通过 `sqlx.NewDb` 包装已有 `*sql.DB`，两者使用同一连接池；同时演示结构体映射、命名参数、IN 切片展开和逐行处理。 对照项目：[`01-database-sql-demo`](../01-database-sql-demo)。

## 文件

```text
02-sqlx-demo/
├── main.go                     # 演示步骤与输出、-init
├── mysql.go                    # 本机连接配置、Open、连接池、PingContext
├── model.go                    # Person、db 标签、sql.NullString
├── person.go                   # 数据操作
├── schema.go                   # -init 执行的建表与初始数据
├── schema.sql                  # 同样的 SQL，可手动执行
├── person_integration_test.go  # 惰性建连及本机 MySQL 集成测试
├── go.mod / go.sum
└── Makefile
```

## 准备本机 MySQL

使用 Go 1.26.5 和已经启动的本机 MySQL。连接地址固定为 `127.0.0.1:3306`。

先使用自己的本机账号登录，创建学习数据库：

```bash
mysql -u root -p
```

```sql
CREATE DATABASE IF NOT EXISTS go_mysql_demo CHARACTER SET utf8mb4;
```

在运行示例的终端设置账号、密码和库名：

```bash
export MYSQL_USER='root'
export MYSQL_PASSWORD='替换为你的本机密码'
export MYSQL_DATABASE='go_mysql_demo'
make init
make run
```

`MYSQL_USER` 默认是 `root`，`MYSQL_DATABASE` 默认是 `go_mysql_demo`；密码从 `MYSQL_PASSWORD` 读取，代码没有硬编码密码。连接配置由 MySQL 驱动的 `Config.FormatDSN` 生成。

`make init` 等同于 `go run . -init`，只创建 person 表并补充 alice、bob 两条初始记录。重复初始化不重置已有记录。两个项目使用相同表结构，可以共用这个学习数据库。

Makefile 的 GO/GOROOT 默认指向本机 Go 1.26.5 SDK，可以通过 make 参数覆盖。直接使用 `go run .` 时，应确保 go、GOROOT 与工具链版本一致。

## 观察结果

程序按顺序执行：

1. 打印 Open 后和 Ping 后的连接池状态，观察实际连接何时建立。
2. 插入带动态后缀的 charlie 用户，查询并更新其年龄。
3. 查询 bob，观察 SQL NULL 对应 `Email.Valid == false`。
4. 查询多行，再按 ID 切片查询并逐行处理结果。
5. 在事务中两次插入同一用户名；确认 MySQL 1062 错误，并检查第一次插入也已回滚。
6. 删除本次 charlie 用户，确认再次查询得到 `sql.ErrNoRows`。

用户名后缀、自增 ID 和记录数量会随运行变化；回滚也可能消耗自增编号。

## 测试

```bash
make test
# 沿用上面设置的本机账号和密码，执行真实数据库测试：
MYSQL_INTEGRATION=1 make test
```

未设置 `MYSQL_INTEGRATION=1` 时，集成测试会跳过；惰性建连测试不依赖 MySQL 服务。

集成测试连接本机 3306，要求账号有创建和删除数据库的权限；每次创建独立的 `go_person_demo_test_*` 库，结束时删除。测试不清空 `MYSQL_DATABASE` 指定的学习库。

测试覆盖 CRUD、NULL 与空字符串、缺失记录、同值更新、单连接池等待和 Context 超时、事务提交与回滚、完整演示流程、IN 切片展开、逐行处理提前返回后释放结果集。
