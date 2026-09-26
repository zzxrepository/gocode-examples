# Kafka 存储：JAVA 订单事件实验

本目录为独立项目，复制本目录即可运行，不引用其他章节源码。

环境：JDK 17+、Maven 3.9.x、Kafka 4.3.1、Python 3（测试）。

```bash
export KAFKA_TOPIC="orders.c06.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-java"
./run.sh init
./run.sh produce --orders 3
./run.sh consume --max 6
python3 scripts/smoke.py
```

`config.example.env` 列出环境变量，命令行参数优先。`--orders` 每笔产生两条事件；`--max 0` 在超时前持续消费；`--prefix` 控制订单 ID；`--duplicate` 重复发送同一事件；`--delay` 模拟处理延迟。重复完整实验应使用新 Topic 和新消费组。

`run.sh` 保证 Maven 和 Java 使用同一个 JDK；设置 `JAVA_HOME` 可覆盖 macOS Homebrew 默认选择。

## 前言

订单事件日志记录发生过什么，订单状态表记录当前是什么。两者适合不同的 Kafka 清理策略：事件流通常保留一段时间的历史，状态变更流可以按 Key 压缩。本实验观察同一订单的两条记录、压缩配置和删除标记。

## 分区日志与段文件

每个分区在 Broker 上由日志段保存，包含记录批次以及 Offset、时间索引等辅助文件。追加写与操作系统页缓存有助于吞吐，索引缩小定位范围。持久化也需要考虑复制、操作系统和磁盘故障；`acks=all` 不能简单解释为每条记录已经在所有磁盘上完成 fsync。

分区副本各自保存日志。保留或压缩不会把剩余记录重新编号，Offset 可能不连续。事务控制记录也会占用日志位置。

## delete 与 compact

| 策略 | 含义 | 订单用途 |
| --- | --- | --- |
| `delete` | 按时间、大小等策略删除符合条件的日志段 | 订单历史事件 |
| `compact` | 后台清理旧值，保留各 Key 的最新值及必要记录 | 订单当前状态变更 |
| `compact,delete` | 同时受两类清理规则约束 | 有保留期的状态日志 |

保留不是消费确认机制：所有组读完不会立即删除，某个组一直不读也不会阻止过期清理。`retention.bytes` 按分区设置，不能直接当作整个 Topic 的容量上限。

压缩异步执行，受段滚动、脏数据比例、清理延迟和 cleaner 负载影响。发送两条相同 Key 的消息后，立即消费仍看到两条是正常的。Kafka 保留的是日志中最后写入的值，并不了解 JSON 内的业务版本；旧业务状态如果后写入，也可能成为压缩后保留的值。

## 本章状态日志

`init` 将本章 Topic 的 `cleanup.policy` 设为 compact。创建和支付事件使用同一订单 Key；`snapshot` 从消息重建内存状态，并根据订单版本避免较旧版本覆盖较新版本。

这是有限实验中的状态投影，内存状态不跨进程保存。重新构建必须使用新组从当前保留数据开头读取，不能沿用已有提交位置后期待自动恢复全部状态。

```bash
export KAFKA_TOPIC="orders.c06.$(date +%s)"
./run.sh init
./run.sh produce --orders 3
./run.sh inspect
./run.sh snapshot --group "$KAFKA_TOPIC-rebuild" --max 6
```

输出应有三条 `STATE`，每条版本为 2。若该 Topic 已被后台压缩，记录数可能小于六条，此时改用 `--max 0 --timeout 10s` 进行有时限的读取，不把记录数写死。

## Tombstone

带 Key、Value 为 null 的记录称为 tombstone，用来表达该 Key 被删除。JSON 字符串 `"null"`、空对象 `{}` 和 Kafka null Value 不是同一个东西。

```bash
# 删除 order-1 对应的状态；其余订单保留。
./run.sh tombstone --prefix order
./run.sh snapshot --group "$KAFKA_TOPIC-rebuild-after-delete" --max 0 --timeout 10s
```

新快照应只剩两个订单。Tombstone 的物理清理同样有延迟；长时间离线的状态消费者需要考虑删除标记的保留期限，否则可能错过删除事件。应用自己的内存删除行为不等于磁盘上的段文件立即消失。

## 查看实际配置

```bash
kafka-configs --bootstrap-server 127.0.0.1:9092 \
  --describe --entity-type topics --entity-name "$KAFKA_TOPIC"
kafka-topics --bootstrap-server 127.0.0.1:9092 \
  --describe --topic "$KAFKA_TOPIC"
python3 scripts/smoke.py
```

`inspect` 输出的 start 是最早仍可读取位置，end 是日志末尾位置。`end-start` 不能总是等于可见业务记录条数，事务和压缩都可能造成差异。

## 观察磁盘文件

先从所用 Broker 的配置读取 `log.dirs`，不要照抄其他机器的 `/tmp/kafka-logs`。集群实验项目的三节点数据位于该项目的 `.cache/cluster`，Homebrew 默认目录也可能被用户修改。

日志段应通过 Kafka 工具读取，不直接编辑 `.log` 或删除正在使用的分区文件。清理实验数据可以删除明确命名的测试 Topic；停止服务不会删除数据。

## 总结

Kafka 存储的基本单位是分区日志，清理单位和时机由保留、段滚动与压缩机制共同决定。事件日志和最新状态日志要采用不同的清理设计，消费者不能依赖 Offset 连续或消息永久存在。

## 协议与限制

消息使用 JSON，Key 是 UTF-8 订单 ID，两种客户端统一使用 CRC32 取模。生产者使用幂等和 all 确认；消费者使用 classic 消费组、手动提交、read_committed。入门消费者逐条提交并串行处理，不作为吞吐量最佳实践。

原始技术文档来源：[dunwu](https://dunwu.github.io/bigdata-tutorial/kafka/)，本文档改编采用 CC BY-SA 4.0。新增代码采用 MIT 许可。
