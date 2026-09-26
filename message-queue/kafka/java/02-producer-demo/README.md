# Kafka 生产者：JAVA 订单事件实验

本目录为独立项目，复制本目录即可运行，不引用其他章节源码。

环境：JDK 17+、Maven 3.9.x、Kafka 4.3.1、Python 3（测试）。

```bash
export KAFKA_TOPIC="orders.c02.$(date +%s)"
export KAFKA_GROUP="$KAFKA_TOPIC-java"
./run.sh init
./run.sh produce --orders 3
./run.sh consume --max 6
python3 scripts/smoke.py
```

`config.example.env` 列出环境变量，命令行参数优先。`--orders` 每笔产生两条事件；`--max 0` 在超时前持续消费；`--prefix` 控制订单 ID；`--duplicate` 重复发送同一事件；`--delay` 模拟处理延迟。重复完整实验应使用新 Topic 和新消费组。

`run.sh` 保证 Maven 和 Java 使用同一个 JDK；设置 `JAVA_HOME` 可覆盖 macOS Homebrew 默认选择。

## 前言

订单事件进入生产者进程，不代表已经写入 Kafka。序列化、分区、缓冲、网络请求、Broker 确认和应用退出都可能影响最终结果。这个实验用同一批订单比较同步等待与异步回调，并检查它们是否都正确处理成功和失败。

## 发送路径

事件先编码成 JSON，根据订单 Key 选择分区，再进入该分区的批次缓冲；后台发送线程把批次发往分区 Leader，重试可恢复错误，最后通过 Future 或响应通道通知应用。

`bootstrap.servers` 是发现集群的入口地址，不要求配置所有 Broker。客户端取得元数据后直接访问对应分区 Leader。连接和元数据会在后台维护，不应为每条消息重新创建一个生产者。

## 同步和异步的区别

Java 的 `send(record)` 本身返回 Future，调用 `.get()` 才等待该消息的结果；回调可以在发送完成后处理错误。Sarama 提供 `SyncProducer` 和 `AsyncProducer` 两种接口。

同步等待更容易编写，但逐条等待会减少流水线并发。异步方式必须限制在途工作量、处理错误，并在退出前排空已提交的发送任务。**可靠性由配置和错误处理决定，不能用“同步必不丢、异步必丢”概括。**

Go 的异步实现使用后台协程持续读取成功和失败通道。发送循环为每条消息增加待确认计数，收到任一种结果后递减：

```go
// 发送循环中的片段；完整函数包含响应通道的排空协程。
acknowledgements.Add(1)
p.Input() <- m

// 全部发送任务加入后，等待结果，再关闭并等待通道退出。
acknowledgements.Wait()
p.AsyncClose()
return <-done
```

完整实现位于 `main.go` 的 `produce` 函数，关闭任一通道后会将其设为 nil，避免反复读取已关闭通道。Java 实现在回调中保留首次异常，调用 `flush()` 后检查异常；`flush()` 本身不能替代业务侧检查结果。

## 确认、幂等和重试

| 参数 | Java 示例 | Sarama 示例 |
| --- | --- | --- |
| 确认级别 | `acks=all` | `RequiredAcks=WaitForAll` |
| 生产者幂等 | `enable.idempotence=true` | `Idempotent=true` |
| 在途请求 | `max.in.flight.requests.per.connection=1` | `Net.MaxOpenRequests=1` |
| 压缩 | `compression.type=snappy` | `CompressionSnappy` |
| 失败边界 | `delivery.timeout.ms=30000` | 显式网络超时与有限重试 |

Java 幂等生产者允许的在途请求上限是 5；Sarama 此版本要求设为 1。本示例统一取 1 便于解释，不能把这个实验选择写成所有客户端的协议限制。

`acks=0` 不等待 Broker 确认；`acks=1` 等待 Leader；`acks=all` 等待当前 ISR 的确认。`min.insync.replicas` 是允许成功写入的同步副本门槛，不能理解成“只等待指定数量副本”。三副本、ISR 三个、min ISR 两个时，正常 `acks=all` 写入仍等待三个 ISR。

生产者幂等主要消除客户端内部重试造成的重复。业务代码显式调用两次 `send` 仍然是两条记录，不能因此省掉业务 `event_id` 和消费端幂等。

网络异常可能意味着“Broker 已写入，但确认没有回来”。调用方要区分可重试失败、不可重试失败和结果不确定，并保留原事件身份；不能遇到异常就生成一个新事件 ID。

## 批处理和压缩

批次按分区组织，`batch.size` 控制目标批次大小，`linger.ms` 提供短暂等待以聚合更多消息。Kafka Java 4.x 的 linger 默认已不是旧教程常见的 0，本实验显式设置 5ms。压缩作用于批次，同样吞吐量下 Key 分布、消息大小、分区数都会影响效果。

入门程序为正确性逐条展示确认，不把六条小消息的耗时当成吞吐量基准。性能测试需要固定消息大小、负载、批次参数、压缩算法、确认级别和网络环境。

## 运行实验

进入 `go/02-producer-demo` 或 `java/02-producer-demo`，执行：

```bash
export KAFKA_TOPIC="orders.c02.$(date +%s)"
./run.sh init
./run.sh produce --mode sync --orders 3 --prefix sync
./run.sh produce --mode async --orders 3 --prefix async
./run.sh consume --group "$KAFKA_TOPIC-reader" --max 12
```

应收到十二条记录，其中六条来自同步发送、六条来自异步发送。异步 ACK 的打印先后不代表全局业务顺序。

再使用另一个 Topic，执行 `produce --duplicate --orders 3` 并消费十二条记录。虽然生产者幂等已开启，仍会看到相同 `event_id` 出现两次，因为这是应用主动重复发送。

```bash
# 指向不存在的服务，检查命令应返回非零退出码。
./run.sh produce --brokers 127.0.0.1:1 --orders 1

# 本章自动检查异步发送与最终业务记录。
python3 scripts/smoke.py
```

程序不会把加入本地队列打印成发送成功。所有 ACK 都来自生产者成功结果，错误会传播到进程退出码。

## 分区与顺序

两套实现显式使用 CRC32 分区。Key 不变、分区数不变、同一订单串行发送，才能使示例中创建事件先于支付事件进入同一分区。扩分区会改变取模结果；多生产者并发、异步业务处理也可能破坏状态更新次序，应结合订单版本检查。

## 总结

发送可靠性需要同时审视确认级别、ISR、幂等约束、失败处理和程序退出过程。同步与异步是等待结果的方式，不能作为消息可靠性的简单标签。

## 协议与限制

消息使用 JSON，Key 是 UTF-8 订单 ID，两种客户端统一使用 CRC32 取模。生产者使用幂等和 all 确认；消费者使用 classic 消费组、手动提交、read_committed。入门消费者逐条提交并串行处理，不作为吞吐量最佳实践。

原始技术文档来源：[dunwu](https://dunwu.github.io/bigdata-tutorial/kafka/)，本文档改编采用 CC BY-SA 4.0。新增代码采用 MIT 许可。
