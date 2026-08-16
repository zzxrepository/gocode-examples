# gRPC教程-grpc-gateway

> 书店场景：浏览器和第三方系统用 HTTP/JSON 调用图书接口，内部订单服务仍可使用高效的 gRPC 接口。

这个示例演示 grpc-gateway：同一个 gRPC 服务既可以通过 gRPC 客户端调用，也可以通过 HTTP JSON 调用。

原仓库使用 grpc-gateway v1 和同端口复用。这里更新为 grpc-gateway v2，并拆成：

- gRPC TLS 服务：`localhost:8000`
- HTTP gateway：`localhost:8080`

## 本节先懂这些

**gRPC-Gateway 是 HTTP/JSON 和 gRPC 之间的翻译层。**浏览器、`curl`、许多第三方系统更容易发送 HTTP/JSON；内部 Go 服务更适合直接使用 gRPC。Gateway 接到 `POST /v1/books/...` 这类 HTTP 请求后，按 `.proto` 中的映射把 JSON 转成 Proto 请求，再调用后面的 gRPC 服务；响应则反向转回 JSON。

```text
浏览器 / 第三方 ─HTTP JSON→ Gateway ─gRPC/Proto→ 图书服务
内部订单服务 ───────────────────────gRPC/Proto→ 图书服务
```

它的价值是“一份服务定义，提供两种入口”，而不是让 HTTP 和 gRPC 互相替代。业务逻辑仍只应写在 gRPC 服务实现中；Gateway 负责协议适配、HTTP 状态码和 JSON 映射。认证也不能因为多了 Gateway 就省略：HTTP 侧的 `Authorization` 必须安全地转为后端所需的 metadata，并由后端继续校验。

当 HTTP API 对外发布后，路径和 JSON 字段同样属于兼容性契约；修改它们要像修改 Proto 字段一样谨慎。

## 生成 proto 代码

```bash
VALIDATOR_DIR="$(go env GOPATH)/pkg/mod/github.com/mwitkow/go-proto-validators@v0.3.2"
GOOGLEAPIS_DIR="$(go env GOPATH)/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.16.0/third_party/googleapis"

protoc -I . -I "$VALIDATOR_DIR" -I "$GOOGLEAPIS_DIR" \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  --govalidators_out=. --govalidators_opt=paths=source_relative \
  --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
  --openapiv2_out=proto --openapiv2_opt=allow_merge=true,merge_file_name=simple \
  proto/simple.proto
```

## 运行 gRPC 和 HTTP gateway

```bash
go run ./cmd/server
```

gRPC 客户端：

```bash
go run ./cmd/client
```

HTTP 调用：

```bash
curl -X POST http://localhost:8080/v1/example/route \
  -H 'Authorization: bearer grpc.auth.token' \
  -H 'Content-Type: application/json' \
  -d '{"some_integer":99,"some_float":0.5}'
```
