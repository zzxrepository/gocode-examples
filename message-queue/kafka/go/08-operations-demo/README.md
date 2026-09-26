# Kafka 运维：GO 订单事件实验

本目录为独立项目，复制本目录即可运行，不引用其他章节源码。

环境：Go 1.25+、Kafka 4.3.1、Python 3（测试）。

```bash
export KAFKA_TOPIC="orders.c08.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-go"
./run.sh init
./run.sh produce --orders 3
./run.sh consume --max 6
python3 scripts/smoke.py
```

`config.example.env` 列出环境变量，命令行参数优先。`--orders` 每笔产生两条事件；`--max 0` 在超时前持续消费；`--prefix` 控制订单 ID；`--duplicate` 重复发送同一事件；`--delay` 模拟处理延迟。重复完整实验应使用新 Topic 和新消费组。

`run.sh` 为本次进程启用自动 Go 工具链、官方模块代理和校验，不修改全局设置；`KAFKA_GO_PROXY` 可覆盖代理，`GO` 可指定 Go 可执行文件。

## 前言

订单消费者没有报错，也可能已经落后十万条；Broker 正常监听端口，也可能有分区离线。运维需要从业务处理、消费者组、分区副本和基础资源多个层次观察，而不是只检查进程是否存在。

## 观察消费积压

对有提交位置的分区，常见 Lag 近似为日志末尾位置减去该组提交位置。它反映位置差距，不直接等于订单数；事务控制记录、压缩和过滤会造成差异。没有提交位置时应显示 unknown，而不是编造一个精确 Lag。

本章 `lag` 使用管理 API 查询组位置与分区末尾，Go 和 Java 输出相同字段。`inspect` 观察分区 Leader、副本、ISR 及日志范围。

```bash
export KAFKA_TOPIC="orders.c08.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-worker"
./run.sh init
./run.sh produce --orders 10

# 先处理两条，建立提交位置。
./run.sh consume --max 2
./run.sh lag
./run.sh inspect
```

部分尚未提交位置的分区可能显示 unknown。已经提交的分区可以看到 Lag，剩余工作要结合全部分区检查。

```bash
# 处理完剩余十八条后，普通非事务 Topic 的各分区 Lag 应归零。
./run.sh consume --max 18
./run.sh lag
```

## 制造慢消费者

```bash
./run.sh produce --orders 30 --prefix backlog
./run.sh consume --max 60 --delay 200ms --timeout 1m
```

另一个终端反复执行 `lag`，观察积压下降。`--delay` 模拟业务处理耗时，不代表 Kafka 拉取性能。

增加同组实例只在尚有可分配分区时有效。某个订单 Key 形成热点、下游数据库变慢、错误重试阻塞、分区数不足都会限制扩容收益。要先判断瓶颈，再决定增加消费者、调整分区还是优化业务处理。

## 常用命令

```bash
# Topic 与副本状态。
kafka-topics --bootstrap-server 127.0.0.1:9092 \
  --describe --topic "$KAFKA_TOPIC"

# 消费组位置、Lag 和成员信息。
kafka-consumer-groups --bootstrap-server 127.0.0.1:9092 \
  --describe --group "$KAFKA_GROUP"

# Topic 的覆盖配置。
kafka-configs --bootstrap-server 127.0.0.1:9092 \
  --describe --entity-type topics --entity-name "$KAFKA_TOPIC"
```

命令中的 bootstrap 地址是客户端入口，不是 ZooKeeper 地址。KRaft 元数据检查使用 `kafka-metadata-quorum`，不再使用 ZooKeeper shell 管理 Kafka 4.x 元数据。

## 重置消费位置

重置前先停止该组所有成员，确定重放范围和下游幂等性。先预览，再执行，避免把 dry-run 当成已生效。

```bash
kafka-consumer-groups --bootstrap-server 127.0.0.1:9092 \
  --group "$KAFKA_GROUP" --topic "$KAFKA_TOPIC" \
  --reset-offsets --to-earliest --dry-run

# 确认上一步显示的目标位置后执行。
kafka-consumer-groups --bootstrap-server 127.0.0.1:9092 \
  --group "$KAFKA_GROUP" --topic "$KAFKA_TOPIC" \
  --reset-offsets --to-earliest --execute
```

重置只改变该组进度，不恢复已过期数据，也不撤销已经发送的通知或写入的数据库结果。对无幂等设计的消费者直接重放会产生重复副作用。

## 监控与定位

| 现象 | 优先检查 |
| --- | --- |
| 发送超时 | Broker 地址、advertised.listeners、认证、ISR 门槛、生产端超时 |
| Lag 持续增大 | 业务处理耗时、失败重试、分区热点、消费者数量 |
| 频繁再均衡 | 实例重启、poll 间隔、心跳、成员扩缩容与协议配置 |
| 分区不可用 | Leader、ISR、节点磁盘和 Controller 仲裁状态 |
| 数据过期 | retention 配置、磁盘容量与落后时间 |

应监控请求延迟、错误率、离线分区、低于最小 ISR 的分区、磁盘使用率、消费处理耗时、提交失败与再均衡次数。Kafbat UI 便于人工查看；长期告警通常还需要采集 Broker/JVM 和应用指标。

## 配置与权限

三节点实验脚本使用仅监听回环地址的 PLAINTEXT；已有 Broker 应检查自身的 listeners 配置。远程服务需要根据网络环境配置 TLS、SASL 和 ACL，并正确设置客户端可访问的 advertised.listeners。不能把本机无认证配置直接暴露到公网。

实验脚本从环境读取服务地址与数据库配置，不提交真实凭据。日志记录事件 ID、订单 ID、分区和 Offset，避免把完整业务敏感字段无差别输出。

## 清理与验证

```bash
python3 scripts/smoke.py

# 只删除明确命名、确实不再需要的实验 Topic。
kafka-topics --bootstrap-server 127.0.0.1:9092 \
  --delete --topic "$KAFKA_TOPIC"
```

不能用一条广泛匹配的 rm 命令清理正在运行的 Kafka 数据目录。停止本机 Broker 使用对应服务管理器；停止集群章节的三节点实验使用其独立的 cluster 脚本。

## 总结

消费进度、分区健康和业务结果需要同时观察。重置位置、增加实例、改分区数和删除 Topic 都有不同影响，应通过明确的实验对象、预览结果和可重复验证来管理。

## 协议与限制

消息使用 JSON，Key 是 UTF-8 订单 ID，两种客户端统一使用 CRC32 取模。生产者使用幂等和 all 确认；消费者使用 classic 消费组、手动提交、read_committed。入门消费者逐条提交并串行处理，不作为吞吐量最佳实践。

原始技术文档来源：[dunwu](https://dunwu.github.io/bigdata-tutorial/kafka/)，本文档改编采用 CC BY-SA 4.0。新增代码采用 MIT 许可。
