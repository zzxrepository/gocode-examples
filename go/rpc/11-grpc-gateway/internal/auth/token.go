package auth

import "context"

// Token 给 gRPC 客户端调用附加 bearer token。
// gateway 的 HTTP 调用则直接通过 Authorization 请求头传入 token。
type Token struct {
	Value string
}

// GetRequestMetadata 把 token 写入 gRPC metadata。
func (t *Token) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"authorization": t.Value}, nil
}

// token 需要通过 TLS gRPC 连接发送。
func (t *Token) RequireTransportSecurity() bool {
	return true
}
