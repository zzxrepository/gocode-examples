package auth

import "context"

// Token 实现 credentials.PerRPCCredentials。
// 它负责把 Authorization 信息附加到每次 RPC 的 metadata 里。
type Token struct {
	// Value 的格式是 "bearer grpc.auth.token"。
	// 服务端拦截器会解析 bearer 后面的 token 值。
	Value string
}

// GetRequestMetadata 会在每次 RPC 发出前被 gRPC 调用。
// 返回的 map 会变成 metadata，服务端可以通过 metadata.FromIncomingContext 读取。
func (t *Token) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{"authorization": t.Value}, nil
}

// 返回 true 表示 token 只允许通过 TLS 等安全连接发送。
func (t *Token) RequireTransportSecurity() bool {
	return true
}
