package auth

import "context"

// Token 实现 gRPC 的 credentials.PerRPCCredentials 接口。
//
// 客户端把它传给 grpc.WithPerRPCCredentials 后，
// 每次 RPC 调用都会自动附带 app_id 和 app_secret metadata。
type Token struct {
	// AppID 模拟应用标识。
	AppID string
	// AppSecret 模拟应用密钥。
	AppSecret string
}

// GetRequestMetadata 返回每次 RPC 要附加的 metadata。
// 服务端可以从 metadata.FromIncomingContext(ctx) 中读取这些键值。
func (t *Token) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"app_id":     t.AppID,
		"app_secret": t.AppSecret,
	}, nil
}

// RequireTransportSecurity 返回 true，表示这些认证信息必须在安全传输上发送。
// 因为 app_secret 属于敏感信息，所以应该配合 TLS。
func (t *Token) RequireTransportSecurity() bool {
	return true
}
