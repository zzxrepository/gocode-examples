package interceptor

import (
	"context"
	// strings 用来处理 "bearer xxx" 认证头。
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// validator 是一个小接口。
// protoc-gen-govalidators 会给带校验规则的 message 生成 Validate() error 方法。
// 只要请求实现了这个接口，我们就可以统一调用它。
type validator interface {
	Validate() error
}

// Validate 是字段校验拦截器。
// 它在业务方法前运行，发现请求不合法就直接返回 InvalidArgument。
func Validate(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if v, ok := req.(validator); ok {
		if err := v.Validate(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
	}
	// 请求没有 Validate 方法，或校验通过，则继续执行后续逻辑。
	return handler(ctx, req)
}

// Auth 是 bearer token 认证拦截器。
func Auth(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	token := strings.TrimPrefix(first(md.Get("authorization")), "bearer ")
	if token != "grpc.auth.token" {
		return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
	}
	// 认证通过后才进入业务方法。
	return handler(ctx, req)
}

// first 取 metadata 某个 key 的第一个值。
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
