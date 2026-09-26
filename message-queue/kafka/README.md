# Kafka 订单事件实验

围绕订单创建、支付、状态投影和支付统计学习 Kafka。8 个主题分别提供独立 Go/Java 项目，任何目录都能单独复制、构建和运行。

```text
kafka/
├── go/
│   ├── 01-quickstart-demo/
│   └── … 08-operations-demo/
└── java/
    ├── 01-quickstart-demo/
    └── … 08-operations-demo/
```

| 主题 | 实验内容 | Go | Java |
| --- | --- | --- | --- |
| 01 快速入门 | 事件协议、Topic、跨语言读写 | [Go](go/01-quickstart-demo/) | [Java](java/01-quickstart-demo/) |
| 02 生产者 | 同步与异步发送、确认、幂等和分区 | [Go](go/02-producer-demo/) | [Java](java/02-producer-demo/) |
| 03 消费者 | 消费组、手动提交与再均衡 | [Go](go/03-consumer-demo/) | [Java](java/03-consumer-demo/) |
| 04 集群 | 三节点 KRaft、副本和故障切换 | [Go](go/04-cluster-demo/) | [Java](java/04-cluster-demo/) |
| 05 可靠传输 | 事务、MySQL 幂等与 Outbox | [Go](go/05-reliability-demo/) | [Java](java/05-reliability-demo/) |
| 06 存储 | 压缩 Topic、快照和 Tombstone | [Go](go/06-storage-demo/) | [Java](java/06-storage-demo/) |
| 07 流式处理 | 事件时间聚合与 Java Kafka Streams | [Go](go/07-stream-processing-demo/) | [Java](java/07-stream-processing-demo/) |
| 08 运维 | 消费积压、位点重置和监控 | [Go](go/08-operations-demo/) | [Java](java/08-operations-demo/) |

## 环境

Kafka 4.3.1（KRaft），Go 1.25+ / IBM Sarama 1.60.0，JDK 17+ / kafka-clients 4.3.1 / Maven 3.9.x。验证脚本需要 Python 3 和 Kafka 命令行工具；数据库实验额外需要 MySQL 8.x 与 mysql 命令。

Broker 默认 `127.0.0.1:9092`。单节点事务实验要求 Broker 设置 `transaction.state.log.replication.factor=1`、`transaction.state.log.min.isr=1`；消费组内部 Topic 的副本数也应适合单节点。已有服务不要重新格式化数据目录。

在任意 demo 目录执行 `./run.sh init`、`./run.sh produce`、`./run.sh consume --max 6`。各目录 README 包含完整参数、预期输出和章节实验。每次实验使用新 Topic，避免已有记录和组位点干扰。使用 Go/Java 交叉消费时保持 Topic 相同，消费组按业务目的设置。

Go `run.sh` 为本次构建启用自动工具链和公开模块校验；不修改全局 Go 配置。Java `run.sh` 在 macOS 可选用 Homebrew JDK；可通过 JAVA_HOME 覆盖。首次构建需要网络下载依赖。

## 验证

- 每章：`python3 scripts/smoke.py`，使用随机 Topic 检查真实 Kafka 读写和章节行为。
- 可靠传输：设置 MySQL 连接并执行 schema.sql 后，运行 `python3 scripts/reliability-test.py`，验证提交失败窗口和 Outbox 重投。mysql 命令的连接参数由 MYSQL_TEST_ARGS 提供。
- Java 流处理：`python3 scripts/streams-test.py`，验证事务输出和本地状态丢失后的 changelog 恢复。
- 集群：按 04 目录 README 启动三个独立节点，验证 Leader 切换和最小 ISR 门槛；结束后停止实验节点。

测试只删除自己创建的随机 Topic 和数据库记录；一般运行命令不会自动删除业务 Topic。`.cache`、`target`、日志和本地配置不提交。

已验证环境：macOS ARM64、Kafka 4.3.1、Go 1.25.0、OpenJDK 27、MySQL 8.4.11。16 个基础实验、双语言数据库失败重放、事务隔离、三节点故障切换和 Streams changelog 恢复均通过真实服务验证。Java 编译目标为 17。

## 范围与许可

Go 与 Java 共享 JSON 协议和 CRC32 分区规则；逐条提交用于观察处理边界。Go 内存聚合不提供 Kafka Streams 的状态管理能力。Outbox 示例为单投递进程，不是生产级多实例服务。

教程和 README 以 [dunwu Kafka 教程](https://dunwu.github.io/bigdata-tutorial/kafka/) 为主题基础改编，采用 [CC BY-SA 4.0](https://creativecommons.org/licenses/by-sa/4.0/)。新增程序代码采用 [MIT](LICENSE)。依赖使用各自的许可证。
