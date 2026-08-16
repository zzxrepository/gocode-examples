package auth

import "context"

// Token 实现每次 RPC 自动携带 bearer token 的能力。
type Token struct {
	// Value 示例值："bearer grpc.auth.token"。
	Value string
}

// GetRequestMetadata 返回客户端要发送给服务端的 metadata。
func (t *Token) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"authorization": t.Value}, nil
}

// token 需要通过 TLS 发送，避免明文泄露。
func (t *Token) RequireTransportSecurity() bool {
	return true
}
