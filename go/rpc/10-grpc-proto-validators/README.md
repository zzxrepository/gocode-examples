# gRPC教程-proto validators

> 书店场景：创建订单前先校验 ISBN、数量和会员编号等字段，避免无效请求进入库存或支付逻辑。

这个示例演示 proto 字段校验。原仓库使用 `protoc-gen-govalidators` 生成 `Validate()` 方法，再通过服务端拦截器在进入业务逻辑前校验请求。

## 本节先懂这些

**参数校验是在检查“请求长得对不对”，不是在判断“业务能不能做”。**例如 ISBN 不能为空、购买数量必须大于 0、会员 ID 必须符合格式——这些是格式问题，应尽早拒绝。图书是否存在、库存是否充足、优惠券是否可用则依赖业务数据，属于后续业务判断。

把规则写在 `.proto` 中，再由插件生成 `Validate()`，有三个好处：客户端和服务端看到的是同一份约束；不必在每个 handler 重复 `if`；校验可以统一放进拦截器，在业务方法运行前自动执行。即使前端已经校验，服务端仍必须校验，因为调用者可能是其他服务、脚本或恶意请求。

校验并不能替代认证、授权和业务一致性：合法格式的 `book_id` 也可能不存在；数量是正数也可能超过库存。把这三层分开，错误信息和代码会更清晰。

## 生成 proto 代码

```bash
VALIDATOR_DIR="$(go env GOPATH)/pkg/mod/github.com/mwitkow/go-proto-validators@v0.3.2"

protoc -I . -I "$VALIDATOR_DIR" \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  --govalidators_out=. --govalidators_opt=paths=source_relative \
  proto/simple.proto proto/enum.proto proto/uuid.proto
```

## 运行

```bash
go run ./cmd/server
```

另开终端：

```bash
go run ./cmd/client
```
