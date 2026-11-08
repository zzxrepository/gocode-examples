# Go 数据库操作：GORM

配套文章：[Go 数据库操作：GORM](https://zzxrepository.github.io/gocode/backend/go/frameworks-and-ecosystem/02-data-access/02-gorm/)。

使用 GORM 传统 API 和本机 MySQL，演示连接、模型迁移、CRUD、分页、零值条件与更新、软删除、事务和一对多预加载。各文件平铺在一个 Go 模块中，便于与文章逐段对照。

## 文件与文章对应关系

| 文件 | 内容 | 对应章节 |
| --- | --- | --- |
| `main.go` | 命令行入口、连接池状态、初始化与演示运行 | 2.1、2.14 |
| `mysql.go` | `OpenDB`、本机 DSN、连接池配置 | 2.1 |
| `model.go` | `User`、`Product`、`Order`、关联字段和软删除 | 2.2、2.10、2.11 |
| `schema.go` | `AutoMigrate`，为本地演示创建 InnoDB 表 | 2.3 |
| `user.go` | `GetUser`、`ListUsers`，请求 Context 与分页 | 2.5、2.6、2.9 |
| `order.go` | 条件扣减库存与创建订单，统一使用事务句柄 | 2.10 |
| `demo.go` | 创建本次演示数据、串联步骤、按主键清理 | 2.4、2.8 |
| `demo_crud.go` | 查询、零值更新、原生 SQL 和软删除实验 | 2.5～2.8、2.12 |
| `demo_transaction.go` | 提交、业务错误回滚和 `Preload` | 2.10、2.11 |
| `mysql_test.go` | 配置与无效参数测试，不依赖数据库 | — |
| `gorm_integration_test.go` | 真实 MySQL、并发扣减、回滚、取消及清理验证 | — |

模型中的 `Orders` 从一开始就声明完整，统一参与迁移；文章在关联查询一节解释该字段。

## 准备本机数据库

使用 Go 1.26.5、MySQL 8.x。依赖固定为 GORM v1.31.1、GORM MySQL 方言 v1.6.0 和 MySQL 协议驱动 v1.10.0。

登录本机 MySQL：

```bash
mysql -h 127.0.0.1 -P 3306 -u root -p
```

创建专用学习数据库：

```sql
CREATE DATABASE IF NOT EXISTS gorm_demo CHARACTER SET utf8mb4;
```

在项目目录的终端配置连接信息。可以直接使用文章中的 DSN：

```bash
export MYSQL_DSN='root:你的密码@tcp(127.0.0.1:3306)/gorm_demo?charset=utf8mb4&parseTime=true&loc=Local&timeout=3s&readTimeout=5s&writeTimeout=5s'
```

也可以沿用前两个 Demo 的分项环境变量：

```bash
unset MYSQL_DSN
export MYSQL_USER='root'
export MYSQL_PASSWORD='替换为你的本机密码'
export MYSQL_DATABASE='gorm_demo'
```

`MYSQL_DSN` 优先；未设置时，代码根据分项配置生成 DSN，地址固定为本机 `127.0.0.1:3306`，默认账号为 `root`、数据库名为 `gorm_demo`。源码不保存真实密码。

## 初始化与运行

```bash
make init
make run
```

等同于：

```bash
go run . -init
go run .
```

`-init` 只迁移表，不插入演示数据。普通运行也会检查并迁移表，随后创建本次演示需要的用户和商品。

需要同时观察 SQL：

```bash
make sql
# 等同于 go run . -sql
```

Makefile 默认使用 PATH 中的 `go`。如果使用另一套 SDK，可以通过 `make GO=/path/to/go` 指定，并保证 `GOROOT` 与该 SDK 一致。

程序按以下顺序执行：

1. 保留 GORM 默认的版本探测和自动 Ping，打印打开后的连接池状态。
2. 使用 `Create` 插入一条用户并回填主键，再批量创建两条用户和一个商品。
3. 单行查询、分页查询，对比结构体与 Map 的零值条件，以及安全复用基础查询条件。
4. 对比单字段更新、结构体更新、Map 更新和 `Select` 显式更新零值。
5. 在事务中扣减库存并创建订单；库存不足时拒绝下单。
6. 在另一个事务中执行 INSERT 后返回业务错误，验证记录被回滚。
7. 通过 `Preload("Orders")` 查询用户订单。
8. 执行带软删除条件的 `Raw` 查询和 `Exec` 更新。
9. 软删除用户，验证普通查询、`Unscoped` 和邮箱唯一约束的差别。
10. 先删除本次订单，再物理删除本次用户和商品；已有记录保留。

部分关键输出：

```text
Create: 主键已回填=true；批量创建=2 条
First: name=张三 age=20
零值查询: struct=3 条 map=1 条
Session 条件隔离: 成年=2 未成年=1
结构体更新后: age=20 active=true
Map 更新后: age=0 active=false
Select 更新后: age=0 active=false
事务提交: 创建订单成功，剩余库存=3
库存不足: 拒绝下单，剩余库存仍为 3
事务回滚: INSERT 后返回错误，该记录未保留
Preload: 当前用户关联订单=1 条
软删除: 普通查询未找到；Unscoped 的 DeletedAt.Valid=true
软删除后复用邮箱: MySQL 1062，唯一值仍被原记录占用
清理完成: 仅删除本次演示创建的用户、订单和商品
```

自增 ID、连接池数量、分页总数和原生查询返回数量可能随数据库状态变化。未找到记录和重复邮箱实验中出现对应的 GORM 错误日志，是正在验证的预期结果。

正常结束或演示步骤返回错误时，程序都会尝试清理已经创建的演示数据。强制终止进程或数据库不可达时，清理可能无法完成；重新运行会生成不同邮箱，不要求自增编号连续。

## 测试

不连接数据库的测试：

```bash
make test
```

使用上面配置的本机账号执行真实数据库测试：

```bash
make test-integration
```

该命令等同于：

```bash
MYSQL_INTEGRATION=1 go test -count=1 -race -cover ./...
```

未设置 `MYSQL_INTEGRATION=1` 时，真实数据库测试会跳过。启用后只允许连接本机 `127.0.0.1:3306`，账号需要具有创建和删除测试库的权限。

每个数据库测试使用独立的 `gorm_demo_test_*` 临时库，结束后关闭连接池并删除测试库，不清空 `MYSQL_DATABASE` 或 DSN 指定的学习数据库。

主要验证：

- 默认 `gorm.Open` 已访问数据库、主键和时间字段回填。
- CRUD、分页、零值更新、软删除和原生 SQL 的行为。
- 订单写入违反用户外键后，之前的库存扣减被回滚。
- 8 个并发请求购买 5 件库存，成功订单总量不超过库存。
- 请求取消时保留可由 `errors.Is` 判断的取消错误。
- 连续运行完整演示后，演示数据清理完成，已有数据保持原值。
