# Kafka 流式处理：JAVA 订单事件实验

本目录为独立项目，复制本目录即可运行，不引用其他章节源码。

环境：JDK 17+、Maven 3.9.x、Kafka 4.3.1、Python 3（测试）。

```bash
export KAFKA_TOPIC="orders.c07.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-java"
./run.sh init
./run.sh produce --orders 3
./run.sh consume --max 6
python3 scripts/smoke.py
```

`config.example.env` 列出环境变量，命令行参数优先。`--orders` 每笔产生两条事件；`--max 0` 在超时前持续消费；`--prefix` 控制订单 ID；`--duplicate` 重复发送同一事件；`--delay` 模拟处理延迟。重复完整实验应使用新 Topic 和新消费组。

`run.sh` 保证 Maven 和 Java 使用同一个 JDK；设置 `JAVA_HOME` 可覆盖 macOS Homebrew 默认选择。

## 前言

支付统计需要回答“每一分钟发生了多少笔支付、金额是多少”。如果只按程序收到消息的时刻统计，网络延迟和重放会把历史支付算到当前时间。事件时间、窗口、乱序和状态恢复因此成为流处理的核心问题。

## 时间与窗口

事件时间来自 `occurred_at`，处理时间来自消费者当前时钟。一分钟滚动窗口把事件时间按 UTC 分钟向下取整，范围为 `[开始, 开始+1分钟)`。

乱序事件可能在窗口之后到达。持续流系统需要明确允许迟到的范围、何时输出最终结果，以及过迟记录的处理方式。窗口结果在关闭前可以多次更新，不能把每次更新都当作一个新的最终结算结果。

## 两种实验能力

| 命令 | Go 与 Java 有界聚合 | Java Kafka Streams |
| --- | --- | --- |
| 入口 | `aggregate` | `streams` |
| 状态 | 当前进程内存 | 本地状态存储与 Kafka changelog |
| 时间窗口 | 按事件时间分钟分桶 | 一分钟窗口，30 秒 grace |
| 重启策略 | 新组完整回放后重新计算 | 根据应用 ID 与状态恢复 |
| 运行范围 | 固定条数或固定时长的实验 | 在给定时限内运行持续拓扑 |

Sarama 是 Kafka 客户端，不等于 Kafka Streams。手写 map 聚合能说明窗口计算，但不会自动获得状态恢复、再分区、迟到判定和 exactly-once。两个实验不能被标成相同的框架能力。

## 有界聚合

`aggregate` 只统计 `OrderPaid`，按 UTC 分钟累加笔数和整数金额。它不对重复业务事件去重，因此重复投递会重复计数。这是刻意保留的语义边界，可结合可靠性实验观察。

```bash
export KAFKA_TOPIC="orders.c07.$(date +%s)"
./run.sh init
./run.sh produce --orders 3
./run.sh aggregate --group "$KAFKA_TOPIC-replay" --max 6
```

所有 `WINDOW` 行的 count 合计应为 3，amount_cents 合计应为 3000。如果发送跨越分钟边界，会出现两个窗口，合计仍保持不变。

内存聚合没有检查点。再次运行必须换新 group，从保留数据开头重算；沿用已提交组可能没有输入，不能把零结果当作真实业务统计。数据已经过期也无法凭空恢复，因此这个实验不是生产级统计服务。

## Java Kafka Streams

Java 项目额外包含 `kafka-streams` 依赖，使用以下拓扑：

```text
读取订单事件并提取事件时间
  → 过滤支付事件
  → 以 all 作为聚合 Key
  → 重新分区
  → 一分钟窗口 + 30 秒 grace
  → 累加 count:amount_cents
  → 输出到 <输入 Topic>.paid-totals
```

统一 Key 会将所有支付汇聚到一个分区，适合小规模总量实验，但会限制并行度。实际按商户或门店聚合时可把相应业务字段作为 Key，再设计全局汇总。

```bash
# 在本章 Java demo 中运行，application.id 来自 --group。
./run.sh streams --group "$KAFKA_TOPIC-stats" --timeout 30s --replicas 1

kafka-console-consumer --bootstrap-server 127.0.0.1:9092 \
  --topic "$KAFKA_TOPIC.paid-totals" --from-beginning \
  --command-property isolation.level=read_committed \
  --formatter-property print.key=true
```

输出 Key 是窗口开始时间，Value 是 `笔数:金额分`。同一 Key 可能出现多次累计更新，应取最后一个结果，不把所有更新再次求和。

拓扑设置 `processing.guarantee=exactly_once_v2`，协调 Kafka 范围内的状态更新、输出与输入消费位置。它不消除源端主动发布的重复业务事件，也不把 MySQL 更新纳入 Kafka 事务。

短时实验关闭时显式使用 `CloseOptions` 的 `LEAVE_GROUP`，释放组成员身份，减少紧接着重启时等待旧会话超时的情况。

`application.id` 同时决定消费组及内部 Topic 命名。相同 ID 用于恢复同一应用；新实验使用新 ID；不能把不同逻辑的应用无意间部署成相同 ID。状态目录位于本项目 `.cache/streams/`，同时运行多个实例要为它们使用独立的状态目录或进程环境。

## 迟到与重放

Streams 根据输入记录推进 stream time，而不是仅凭墙上时钟关闭窗口。grace 允许一定范围的乱序，超过范围的记录可能被丢弃并计入相关指标。真实系统应同时监控迟到率、源端时间质量和结果更新策略。

完整重放可能再次写出统计结果，因此输出设计需要考虑覆盖更新、幂等或新结果命名空间。仅把消费者 Offset 调回 earliest，并不能自动清空已有聚合状态。

```bash
python3 scripts/smoke.py
```

`smoke.py` 验证有界聚合的数量与金额。Java 项目另有 `python3 scripts/streams-test.py`：检查 read_committed 输出，删除该次实验的本地状态，再验证 changelog 恢复后的累计金额。脚本只清理自己生成的 Topic 和状态目录。

## 总结

流处理不仅是循环消费加一个 map。事件时间决定归属，窗口和 grace 决定乱序边界，状态与恢复决定重启后的结果，事务语义决定哪些输出对下游可见。

## 协议与限制

消息使用 JSON，Key 是 UTF-8 订单 ID，两种客户端统一使用 CRC32 取模。生产者使用幂等和 all 确认；消费者使用 classic 消费组、手动提交、read_committed。入门消费者逐条提交并串行处理，不作为吞吐量最佳实践。

原始技术文档来源：[dunwu](https://dunwu.github.io/bigdata-tutorial/kafka/)，本文档改编采用 CC BY-SA 4.0。新增代码采用 MIT 许可。
