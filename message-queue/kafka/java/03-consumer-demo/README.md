# Kafka 消费者：JAVA 订单事件实验

本目录为独立项目，复制本目录即可运行，不引用其他章节源码。

环境：JDK 17+、Maven 3.9.x、Kafka 4.3.1、Python 3（测试）。

```bash
export KAFKA_TOPIC="orders.c03.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-java"
./run.sh init
./run.sh produce --orders 3
./run.sh consume --max 6
python3 scripts/smoke.py
```

`config.example.env` 列出环境变量，命令行参数优先。`--orders` 每笔产生两条事件；`--max 0` 在超时前持续消费；`--prefix` 控制订单 ID；`--duplicate` 重复发送同一事件；`--delay` 模拟处理延迟。重复完整实验应使用新 Topic 和新消费组。

`run.sh` 保证 Maven 和 Java 使用同一个 JDK；设置 `JAVA_HOME` 可覆盖 macOS Homebrew 默认选择。

## 前言

订单状态、通知和统计需要分别处理相同事件；同一种业务还可能启动多个实例分担工作。消费者组决定“哪些实例共享一份工作”，分区决定这种共享能够达到的并行度，消费位置决定失败后从哪里继续。

## 组、分区与 Offset

不同 `group.id` 各自读取 Topic。同一经典消费组内，每个分区同一时刻由一个成员消费；一个成员可以负责多个分区。三个分区最多让三个成员同时获得该 Topic 的分区，第 4 个成员可能空闲。

提交位置表示**下一条要消费的 Offset**。处理完分区 0 的 Offset 10，应提交 11。一次 poll 取得 10～19，但只处理完 10 时，不能直接提交整批拉取位置 20。

Java 示例逐条提交已完成记录：

```java
// processor 成功返回后才更新这一分区的位置。
processor.accept(event, record);
consumer.commitSync(Map.of(
    new TopicPartition(record.topic(), record.partition()),
    new OffsetAndMetadata(record.offset() + 1)
));
```

Sarama 使用 `session.MarkMessage` 标记下一位置，再调用 `session.Commit()` 请求提交。Mark 不等于已经持久提交；Sarama 的 Commit API 不直接返回提交错误，失败后仍可能重放，因此业务处理必须允许重复。

## 消费成功与提交成功

- 先提交、后处理：进程在两步之间退出，记录可能对该组被跳过。
- 先处理、后提交：进程在两步之间退出，业务可能再次执行。
- 业务写入和去重标记放进同一个数据库事务，再提交 Kafka 位置：可将重放转化成幂等的业务结果。

自动提交按客户端实现触发，不是“每五秒业务肯定完成一次”。配置周期不能解决业务处理与位点提交之间的原子性。

示例为了清楚展示边界，逐条同步提交，并在 Go 中串行执行处理回调。这会限制吞吐量。实际服务可以按批提交，但必须维护**每个分区连续处理完成的最大位置**；不能让后完成的记录掩盖前面仍在处理或已失败的记录。

## 从头读取与继续读取

`auto.offset.reset=earliest` 只在该组没有有效提交位置或位置失效时生效。已经提交的组会从原位置继续，不会因为配置 earliest 就每次重放。

需要重放时使用新组，或者在所有成员停止后显式重置旧组位置。删除 Topic、删除组和重置 Offset 是不同操作。

## 再均衡与客户端协议

成员加入、离开，或订阅分区变化，可能触发分区重新分配。撤销分区时应停止相关处理，处理好已完成位置和未完成工作，不能让旧任务继续写入已经转交给新成员的状态。

本组实验显式使用 **classic 协议和 Range 分配策略**，便于 Java 与 Sarama 对照。Kafka 4.x 还提供新的 consumer 协议，其心跳、分配及配置语义与经典协议不同；Broker 支持新协议不代表所有客户端都实现了它。有关 session timeout 的默认值也必须结合协议和客户端核对。

Java Consumer 不是线程安全对象，poll、seek 和提交应由所属线程协调。长时间处理可能超过 `max.poll.interval.ms` 导致再均衡；调大超时不能代替合理的批大小和并发设计。

## 运行实验

进入 `go/03-consumer-demo` 或 `java/03-consumer-demo`：

```bash
export KAFKA_TOPIC="orders.c03.$(date +%s)"
./run.sh init --partitions 3
./run.sh produce --orders 3

# 两个组分别收到六条消息。
./run.sh consume --group "$KAFKA_TOPIC-orders" --max 6
./run.sh consume --group "$KAFKA_TOPIC-notifications" --max 6
```

相同组再次运行会等待新消息，超时后报告未达到目标数量。这是已有位点生效，并不是 Kafka 丢失消息。

观察同组分配时，两个终端设置相同 Topic 和 Group，分别执行：

```bash
./run.sh consume --group "$KAFKA_TOPIC-workers" --max 0 --timeout 2m
```

第三个终端持续发送 `./run.sh produce --orders 30 --prefix load`，观察 `ASSIGNED` 和 `REVOKED`。两个消费者合计处理事件；由于调度差异，不保证每个实例恰好处理一半。`--max 0` 表示在指定时限内持续读取，不要求固定条数。

```bash
kafka-consumer-groups --bootstrap-server 127.0.0.1:9092 \
  --describe --group "$KAFKA_TOPIC-workers"
python3 scripts/smoke.py
```

## 异常消息与重试

无法解析的消息会使 demo 返回错误，并保留未提交位置。生产服务可以选择有限重试、暂停分区、隔离异常事件或转移到应用定义的死信 Topic，但必须决定什么时候允许跳过原记录。

Kafka 不会自动把失败消息移到“死信队列”。写入死信 Topic 后提交原位点仍存在原子性问题，可以使用 Kafka 事务协调两个动作，或接受可重复处理并增加幂等设计。把失败订单转入重试 Topic 还可能改变原有顺序。

## 总结

消费组共享分区，提交位置记录下一条读取位置，重放来自处理与提交之间的失败窗口。理解这些边界之后，才能合理设计并发、再均衡和幂等处理。

## 协议与限制

消息使用 JSON，Key 是 UTF-8 订单 ID，两种客户端统一使用 CRC32 取模。生产者使用幂等和 all 确认；消费者使用 classic 消费组、手动提交、read_committed。入门消费者逐条提交并串行处理，不作为吞吐量最佳实践。

原始技术文档来源：[dunwu](https://dunwu.github.io/bigdata-tutorial/kafka/)，本文档改编采用 CC BY-SA 4.0。新增代码采用 MIT 许可。
