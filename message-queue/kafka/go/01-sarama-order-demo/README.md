# Go + Kafka：Sarama 订单事件实验

通过 HTTP 下单，用独立路由比较同步、异步、回调发送；消费者单独运行，观察分区、位移、再均衡和事务隔离。每种发送方式有自己的代码文件，不使用 `-mode` 分发器。

完整教程：[Go 集成 Kafka：用 Sarama 从订单事件走到可靠消费](https://zzxrepository.github.io/gocode/backend/message-queue/kafka/09-go-sarama.html)。

## 环境与配置

- 本地 Kafka 4.x，已在 Kafka 4.3.1 上验证。
- Go 1.25+，IBM Sarama 固定为 v1.60.2，Viper 固定为 v1.21.0。
- 修改 `configs/local.yaml`：`kafka.bootstrap_servers` 默认 `127.0.0.1:9092`，HTTP 默认 `127.0.0.1:18080`。
- `bootstrap_servers` 就是 Java 客户端 `bootstrap.servers` 的对应应用配置，传给 Sarama 构造函数的第一个参数。
- 单机 PLAINTEXT 实验，不需要数据库。订单只保存在 API 进程内存中，重启会丢失。

从 GoTutorials 工作区进入：

```bash
cd gocode-examples/message-queue/kafka/go/01-sarama-order-demo
```

从独立仓库获取：

```bash
git clone https://github.com/zzxrepository/gocode-examples.git
cd gocode-examples/message-queue/kafka/go/01-sarama-order-demo
```

## 启动与调用

```bash
# 创建三分区、一副本 Topic；已存在时保留配置和数据。
sh run.sh ./cmd/init-topic

# 终端 A：订单 API。
sh run.sh ./cmd/api

# 终端 B：消费者组。
sh run.sh ./cmd/consumer
```

`run.sh` 仅为本次命令启用 Go 工具链自动选择、公开模块代理与校验；首次可能下载依赖。Go 环境符合要求时也可直接 `go run ./cmd/api`。

终端 C 依次调用：

```bash
# 同步：HTTP 201 表示已得到 Broker 确认。
curl -i -X POST http://127.0.0.1:18080/demo/sync/orders \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"sync-1001","amount_cents":1200}'
curl -i -X POST http://127.0.0.1:18080/demo/sync/orders/sync-1001/pay
curl -sS http://127.0.0.1:18080/demo/sync/orders/sync-1001

# 异步：HTTP 202 只表示已交给 Sarama，失败结果在 API 日志中观察。
curl -i -X POST http://127.0.0.1:18080/demo/async/orders \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"async-1001","amount_cents":2300}'

# 回调：HTTP 202 后在 API 日志中看到逐条成功或失败结果。
curl -i -X POST http://127.0.0.1:18080/demo/callback/orders \
  -H 'Content-Type: application/json' \
  -d '{"order_id":"callback-1001","amount_cents":3500}'
```

三种路由都支持以下操作，各自维护独立的内存订单集合。请给不同 demo 使用不同订单号；同一订单的后续请求保持同一路由前缀。

| 方法与路径（以同步路径为例） | 含义 |
| --- | --- |
| `POST /demo/sync/orders` | 创建订单，body 含 order_id、amount_cents |
| `GET /demo/sync/orders/{id}` | 查询本进程内存状态 |
| `POST /demo/sync/orders/{id}/pay` | 从 created 变为 paid |
| `POST /demo/sync/orders/{id}/cancel` | 从 created 变为 cancelled |

重复创建或非法状态转换返回 409，不存在返回 404，非法参数返回 400，同步发布未确认返回 503。503 可能是结果未知，不能据此断言 Kafka 一定没有收到。异步接口的本地状态表示请求已受理，不保证 Broker 已保存；后续发送错误不会修改已经返回的 HTTP 202。

## 代码入口

```text
cmd/api/                    HTTP 服务组装和退出
cmd/init-topic/             创建 Topic
cmd/consumer/               消费者组进程
cmd/partition-reader/       指定分区读取 demo
configs/local.yaml          配置
internal/router/            路由绑定
internal/controller/        请求解析、响应和事务实验入口
internal/service/           订单规则、事件编码、内存状态
internal/model/             订单、OrderPayload、事件别名与事务样例
internal/event/             通用 Event[T]，公共元数据 + Payload
internal/messaging/
  message.go                通用 Record，不依赖 OrderEvent
  sync_producer.go          同步发送
  async_producer.go         异步发送与错误回收
  callback_producer.go      异步成功/失败回调
  transaction_producer.go   Kafka 事务
internal/consumer/          分区处理、标记、提交和再均衡生命周期
internal/config/            Viper 加载、校验与 Sarama 配置
integration/                真实 Broker 集成测试
```

业务路径：`router → controller → service → messaging → Kafka`。新增业务事件应由对应业务层定义 schema 并序列化，再复用 `Record{Key, Value, EventID}`；不用为每个 `XXXEvent` 复制发送代码。

## 事件格式

`internal/event.Event[T]` 定义统一信封，订单使用 `Event[model.OrderPayload]`。Kafka Value 为：

```json
{
  "id": "sync-1001:created",
  "type": "order.status_changed",
  "version": 1,
  "source": "order-service",
  "occurred_at": "2026-09-28T02:00:00Z",
  "payload": {
    "order_id": "sync-1001",
    "status": "created",
    "amount_cents": 1200
  }
}
```

新增业务定义自己的 `PaymentPayload`、`UserPayload` 等，再复用 `Event[T]` 与发送接口。`OrderEvent` 只是类型别名，不重复定义公共字段。多类型入口可先用 `Event[json.RawMessage]` 解析元数据，再按类型与版本分发给业务处理器。当前消费者专门处理订单 v1；这是一套自定义格式，不是完整 CloudEvents 实现。

默认 Topic 为 `order-events-v1`。不兼容的历史消息应使用独立 Topic 或显式迁移处理器，避免旧格式阻塞新消费者。

## 消费、重放与失败实验

```bash
# 同组的两个实例分工；不同 client-id 方便观察。
sh run.sh ./cmd/consumer -group order-progress -client-id progress-a
sh run.sh ./cmd/consumer -group order-progress -client-id progress-b

# 另一个组独立读取同一份数据。
sh run.sh ./cmd/consumer -group order-audit

# 有限时长实验。
sh run.sh ./cmd/consumer -group order-replay-demo -duration 10s

# 处理 paid 时失败退出：不提交该条，再去掉参数重启以观察恢复。
sh run.sh ./cmd/consumer -group order-failure-demo -fail-status paid
sh run.sh ./cmd/consumer -group order-failure-demo -duration 10s

# 指定分区，从最早保留位置开始读；Ctrl+C 退出，不提交组位移。
sh run.sh ./cmd/partition-reader -partition 0 -offset -2
```

同组重启从提交位置续读。配置里的 `initial_offset: oldest` 只在没有有效组位移时决定起点。换一个新 Group ID 才是最方便的重放实验方式。

默认处理成功后 `MarkMessage`，再调用 `Commit`。修改配置里的 `auto_commit: true`，可观察定时提交已标记位移；`MarkMessage` 本身不是提交。查看实际位置：

```bash
kafka-consumer-groups --bootstrap-server 127.0.0.1:9092 \
  --describe --group order-progress
```

## 事务实验

为了准确计数，复制配置并使用一个全新的 Topic，随后让 init-topic、API 和消费者都使用同一个 `-config configs/txn.yaml`。先停止原 API，避免占用同一端口。

```bash
curl -i -X POST http://127.0.0.1:18080/demo/transaction/abort
curl -i -X POST http://127.0.0.1:18080/demo/transaction/commit
```

每个接口生成六条样例状态消息。默认 committed 消费者只读到已提交事务的六条。新建组并使用 `-read-uncommitted` 可以读到两批共十二条：

```bash
sh run.sh ./cmd/consumer -config configs/txn.yaml \
  -group txn-inspect -read-uncommitted -duration 10s
```

事务路由在单进程内串行使用 `client_id + "-http-txn"`。同时运行多个 API 实例必须配置各自稳定且唯一的 client_id，避免互相隔离。事务只覆盖 Kafka 写入，不覆盖数据库和外部支付接口。

## 压缩与独立配置

在 YAML 中将 `producer.compression` 改为 `gzip` 后重启 API。消费者仍正常读取 JSON。支持 none、gzip、snappy、lz4、zstd。

所有程序支持 `-config`。Viper 使用独立实例加载 YAML，结构体使用 `mapstructure` 标签；`UnmarshalExact` 拒绝未知字段，之后校验应用配置与 Sarama 参数组合。

覆盖顺序为：默认值 < YAML < 环境变量 < 消费者显式传入的 `-group`、`-client-id`。所有已知键都注册默认值，使 YAML 未包含的字段也能从环境变量解码。环境变量前缀为 `ORDER_DEMO`，配置键中的点替换为下划线：

```bash
ORDER_DEMO_HTTP_ADDRESS=127.0.0.1:18081 sh run.sh ./cmd/api
ORDER_DEMO_KAFKA_BOOTSTRAP_SERVERS=127.0.0.1:9092,127.0.0.1:19092 \
  sh run.sh ./cmd/consumer -group order-audit
```

引导地址列表使用逗号分隔，请填写实际运行的 Broker。也可以通过 `ORDER_DEMO_KAFKA_TOPIC`、`ORDER_DEMO_KAFKA_CONSUMER_AUTO_COMMIT` 等覆盖其他字段。配置文件必须存在；配置变更需重启进程。

## 验证

```bash
export GOTOOLCHAIN=auto GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org
go test ./...
go vet ./...
KAFKA_TEST_BROKER=127.0.0.1:9092 go test -race -v -count=1 -timeout 3m ./...
```

单元测试覆盖配置优先级、环境变量列表解码、未知字段与无效组合，以及不同 Payload 的信封编码与解码。未设置 KAFKA_TEST_BROKER 时跳过 Broker 测试。集成测试验证三种 HTTP 路由到 Kafka 的写入与顺序、GZIP、异步错误回收、组提交/续读/独立消费、失败记录恢复、事务隔离；测试只清理自己创建的 Topic 和组。

## 实验边界

- 本地一副本不具备多 Broker 容灾能力。
- 订单状态只在内存里，进程重启会丢失；发布与内存更新不是持久原子事务。
- 为直观表达状态检查与发布顺序，订单服务持有互斥锁完成发布，会限制不同订单的并发。
- HTTP 事件 ID 使用“订单号:状态”，只适用于此处单向状态机；真实重复状态迁移需要独立事件 ID。
- 幂等生产者不替消费者做业务去重；异步受理后失败只记录日志，未实现持久补偿。
- 消费业务只打印日志。数据库去重、Outbox、多 Broker 故障注入和生产压测需要另行实现。
