# Kafka 可靠传输：GO 订单事件实验

本目录为独立项目，复制本目录即可运行，不引用其他章节源码。

环境：Go 1.25+、Kafka 4.3.1、Python 3（测试）。

```bash
export KAFKA_TOPIC="orders.c05.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-go"
./run.sh init
./run.sh produce --orders 3
./run.sh consume --max 6
python3 scripts/smoke.py
```

`config.example.env` 列出环境变量，命令行参数优先。`--orders` 每笔产生两条事件；`--max 0` 在超时前持续消费；`--prefix` 控制订单 ID；`--duplicate` 重复发送同一事件；`--delay` 模拟处理延迟。重复完整实验应使用新 Topic 和新消费组。

`run.sh` 为本次进程启用自动 Go 工具链、官方模块代理和校验，不修改全局设置；`KAFKA_GO_PROXY` 可覆盖代理，`GO` 可指定 Go 可执行文件。

## 前言

订单处理最危险的地方往往不是正常流程，而是两个系统都正确执行了一半：数据库已更新，但消费位置还没提交；订单已入库，但事件还没发送；Broker 已保存消息，但生产者没收到响应。这些失败窗口必须分别处理。

## 三种语义与适用范围

At-most-once 允许丢失、避免重投；at-least-once 允许重投，需要处理重复；exactly-once 描述的是明确边界内的处理结果。Kafka 事务能够把 Kafka 写入与相关消费位置更新协调起来，但不会自动把 MySQL、HTTP 调用或邮件发送纳入事务。

本章分别实现三个可运行实验：数据库消费幂等、Kafka 事务提交/回滚、数据库 Outbox 投递。它们不是同一个机制，也不能互相替代。

## 数据库消费幂等

`processed_events` 使用 `(consumer_name,event_id)` 唯一键，`order_projection` 保存订单当前状态。去重记录和业务更新位于同一个 MySQL 事务内：

```text
开始数据库事务
  插入已处理事件标记
  更新订单状态和版本
提交数据库事务
提交 Kafka 消费位置
```

插入去重标记遇到唯一键冲突，表示该业务消费者已处理过此事件，可以安全跳过本次业务更新。不能先在 Redis 中标记成功，再独立写数据库；两个动作之间崩溃会留下错误标记。

状态更新使用版本条件避免旧事件覆盖新状态。这个投影是“保留最高版本的当前状态”，不是完整订单状态机；它不能用来直接代替扣款、扣库存等必须严格检查转移前提的操作。

### 建表和连接

```bash
# 使用自己的本机测试账户；SQL 只创建 kafka_order_demo 数据库中的课程表。
mysql -u root -p < schema.sql
```

Go 设置 `MYSQL_DSN`，格式为 `用户名:密码@tcp(127.0.0.1:3306)/kafka_order_demo?parseTime=true`。Java 设置 `MYSQL_URL=jdbc:mysql://127.0.0.1:3306/kafka_order_demo`、`MYSQL_USER` 和 `MYSQL_PASSWORD`。密码通过环境或本地未提交文件提供，不能写进仓库。无密码测试账户使用相应空值。

### 重复与失败窗口

```bash
export KAFKA_TOPIC="orders.c05.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-projector"
./run.sh init
./run.sh produce --duplicate --orders 3

# 在首次数据库提交后故意失败，此时不提交该条 Kafka 位置。
FAIL_AFTER_DB=1 ./run.sh project --max 12

# 解除故障后重放；第一次已成功的数据库更新应被识别为重复。
./run.sh project --max 12
```

没有成功提交 Kafka 位置时，重启仍会看到全部十二条。输出包含 `DUPLICATE`，最终该消费组应有六个去重记录、三个订单投影，订单状态为 `OrderPaid`、版本为 2。`FAIL_AFTER_DB` 必须只用于实验。

```sql
SELECT consumer_name, COUNT(*) FROM processed_events GROUP BY consumer_name;
SELECT consumer_name, order_id, status, version, amount_cents FROM order_projection;
```

网络故障还可能让数据库提交结果不确定。遇到这种情况返回失败并允许重放，依赖唯一键恢复业务结果；不能把异常一概解释为“数据库肯定没执行”。

设置连接后运行 `MYSQL_TEST_ARGS="--login-path=kafka-course" python3 scripts/reliability-test.py` 可自动检查数据库提交和 Outbox 发送两个失败窗口。需先用 mysql_config_editor 配置 kafka-course 登录项。`MYSQL_TEST_ARGS` 是 mysql 命令行的连接参数，Go/Java 程序仍使用各自的 MYSQL 环境变量；自动化场景推荐使用 MySQL login-path。

## Kafka 事务实验

两个客户端提供 `transaction` 命令，将多条订单事件放入同一个 Kafka 事务。消费端均配置 `read_committed`，因此不会读取已回滚事务中的记录。

```bash
# 使用新 Topic，避免和上面的数据库实验混合。
export KAFKA_TOPIC="orders.tx.$(date +%s)"
./run.sh init
./run.sh transaction --abort --prefix aborted --orders 3
./run.sh transaction --prefix committed --orders 3
./run.sh consume --group "$KAFKA_TOPIC-reader" --max 6
```

只应读到 committed 前缀的六条记录。事务 Topic 还包含控制记录，因此日志 Offset 可能有空洞，不能用末尾 Offset 直接当作业务消息数量。

`TRANSACTIONAL_ID` 标识一个逻辑事务生产者，重启时可沿用，两个并发实例不能随意共用，否则会产生 fencing。单节点实验需将 Broker 的 `transaction.state.log.replication.factor`、`transaction.state.log.min.isr` 配置为 1；生产环境应按实际冗余目标配置。

Sarama 初始化遇到 `ConcurrentTransactions` 时仅对这个瞬态错误做最多五秒的重试；其他错误直接返回，不把 fencing 或配置错误无限重试。

这个命令验证的是**事务写入可见性**，没有执行消费、处理、再写出的全链路事务。实现那种链路时还需把输入消费组位置加入同一事务，并正确处理中断与再均衡。仅调用 begin/commit 不能推出端到端 exactly-once。

## Outbox：订单入库与事件生成

`place` 在同一个 MySQL 事务中更新 `source_orders`、写入 `order_outbox`。`relay` 批量取出尚未发送的事件，收到 Kafka ACK 后标记 `published=1`。

```bash
export KAFKA_TOPIC="orders.outbox.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-projector"
./run.sh init
./run.sh place --orders 3 --prefix "outbox-$(date +%s)"

# 模拟已发送但未标记：命令返回失败，数据库仍保留待投递事件。
FAIL_AFTER_SEND=1 ./run.sh relay
./run.sh relay

# 六个业务事件加一次重复投递，共七条 Kafka 记录。
./run.sh project --max 7
```

Outbox 解决订单提交和“待发送事实”之间的原子性。它仍然可能重复投递，消费端幂等仍然必需。示例 relay 每次最多发送 100 条，运行一次退出，适合查看失败窗口；生产服务还要增加循环调度、重试退避、批次租约或锁、监控与归档。

本实现只演示单 relay 进程，不能把它当作已支持多实例抢占的投递服务。两个并发 relay 可能选择相同记录，带来更多重复。重新执行 place 需使用新的订单前缀，已有业务主键冲突会回滚。

## 无法处理的事件

有限重试解决暂时故障，不能修复永久错误。错误事件应记录原因并提供人工处理或重放入口。死信 Topic、重试 Topic 是应用设计，不是 Kafka 自动提供的处理保证。跨 Topic 重试可能改变订单先后顺序，应结合版本与业务规则决定是否允许。

## 总结

生产者幂等处理内部重试，Kafka 事务处理 Kafka 范围内的原子操作，数据库幂等处理业务重放，Outbox 处理业务提交与待投递事件的一致性。可靠性来自这些机制各自承担清晰责任，而不是一个“绝不丢失”的配置开关。

## 协议与限制

消息使用 JSON，Key 是 UTF-8 订单 ID，两种客户端统一使用 CRC32 取模。生产者使用幂等和 all 确认；消费者使用 classic 消费组、手动提交、read_committed。入门消费者逐条提交并串行处理，不作为吞吐量最佳实践。

原始技术文档来源：[dunwu](https://dunwu.github.io/bigdata-tutorial/kafka/)，本文档改编采用 CC BY-SA 4.0。新增代码采用 MIT 许可。
