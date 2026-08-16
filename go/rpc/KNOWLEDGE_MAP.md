# RPC 知识地图：以书店下单为例

假设 `OrderService.CreateOrder` 需要依次读取图书详情和库存、校验优惠券、读取会员权益，最后创建订单。它不应知道这些服务部署在哪台机器上，也不应无限等待任一服务。

```text
CreateOrder
  ├─ Catalog.GetBook        图书是否存在、价格是多少
  ├─ Inventory.BatchGet     是否有库存
  ├─ Coupon.Validate        优惠券能否使用
  └─ Member.GetBenefits     会员等级和积分
```

| 知识点 | 对应章节 | 在实际订单服务中的含义 |
| --- | --- | --- |
| RPC 最小闭环 | `01-native-go-rpc` | 服务端注册方法，客户端通过方法名、参数和结果完成远程调用 |
| 序列化与接口契约 | `01`、`02-protobuf-basics` | 双方必须约定 `BookRequest`、`BookInfo` 的字段；Go 的 gob 与 Proto 是不同编码方案 |
| 一元调用 | `03-grpc-unary` | 查一本书：一问一答，是绝大多数内部服务调用的形态 |
| deadline 与取消 | `04-grpc-deadlines` | 库存超时就尽快结束，释放连接和下游资源，不能把超时层层放大 |
| 服务端流 | `05-grpc-server-stream` | 搜索/导出时，服务端持续返回一批批图书结果 |
| 客户端流 | `06-grpc-client-stream` | 门店持续上报盘点记录，完成后才收到汇总结果 |
| 双向流 | `07-grpc-bidirectional-stream` | 客服和库存系统持续双向交换消息 |
| 错误模型 | `04`、`03` | 区分网络错误、超时、gRPC status code 和业务错误码；HTTP 200 不等于业务成功 |
| 安全 | `08`、`09` | TLS 保护传输；token/metadata 让库存服务确认调用方身份 |
| 统一横切逻辑 | `08-grpc-middleware` | 日志、trace、认证、panic recovery 放拦截器，业务方法只保留业务逻辑 |
| 参数校验 | `10-grpc-proto-validators` | 空 ISBN、非法数量等请求先被拒绝，不进入创建订单逻辑 |
| HTTP 兼容层 | `11-grpc-gateway` | 网页和第三方仍可用 HTTP/JSON，内部服务可继续使用 gRPC |
| 服务发现与负载均衡 | `12-dirpc-multi-protocol` | `disf!book-inventory` 解析到多个实例；业务代码不保存 IP 地址 |
| 连接复用 | `03`、`12` | client/connection 应在进程内复用，不为每笔订单重复建连 |
| Trace 与 metadata/header | `02`、`08`、`12` | 用同一个 trace ID 串起订单、优惠券、库存的全部日志 |
| 限流、熔断与降级 | `12` | 高峰时保护会员/库存下游；是否重试或降级取决于具体业务语义 |

## 生产环境中还要自己做的判断

框架能提供超时、限流和重试能力，但不能替业务决定策略。下单这类写操作通常要特别注意：

- 只对**幂等**操作谨慎重试；创建订单应传入幂等键，避免超时后重复下单。
- deadline 要沿调用链向下传递，不要每一层都重新给一个更长的超时。
- 读取场景可考虑缓存和降级；扣库存、支付等强一致场景不能随意返回旧数据。
- 用 trace ID、指标和结构化日志定位慢调用、错误分布和热点下游。
- 修改 Proto/IDL 时坚持向后兼容：不要复用已发布字段号，不要随意改变字段语义。
