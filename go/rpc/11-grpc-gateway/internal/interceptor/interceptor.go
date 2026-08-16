package interceptor

import (
	"context"
	// strings 用来处理 bearer token 前缀。
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// validator 是 protoc-gen-govalidators 生成代码会满足的接口。
type validator interface {
	Validate() error
}

// Validate 在进入业务方法前执行字段校验。
// 对 HTTP gateway 请求同样生效，因为 gateway 最终也会转成 gRPC 调用。
func Validate(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if v, ok := req.(validator); ok {
		if err := v.Validate(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
	}
	return handler(ctx, req)
}

// Auth 校验 Authorization metadata。
//
// gRPC 客户端通过 PerRPCCredentials 写入 metadata；
// HTTP gateway 请求通过 Authorization 请求头进入，再由 gateway 转成 metadata。
func Auth(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}
	token := strings.TrimPrefix(first(md.Get("authorization")), "bearer ")
	if token != "grpc.auth.token" {
		return nil, status.Error(codes.Unauthenticated, "invalid bearer token")
	}
	return handler(ctx, req)
}

// first 获取 metadata 某个 key 的第一个值。
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
