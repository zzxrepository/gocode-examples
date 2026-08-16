package main

import (
	// context 会携带客户端传来的 metadata。
	"context"
	"log"
	"net"

	pb "go-grpc-security/proto"

	"google.golang.org/grpc"
	// codes/status 用来返回标准 gRPC 错误码。
	"google.golang.org/grpc/codes"
	// credentials 用来加载服务端 TLS 证书。
	"google.golang.org/grpc/credentials"
	// metadata 用来读取客户端随 RPC 附带的 app_id/app_secret。
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	address = ":8000"
	network = "tcp"
)

type simpleService struct {
	pb.UnimplementedSimpleServer
}

// Route 是受保护的业务方法。
// token 校验已经放到 authInterceptor 中，这里只关心业务响应。
func (s *simpleService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	return &pb.SimpleResponse{
		Code:  200,
		Value: "hello " + req.GetData(),
	}, nil
}

func main() {
	// 监听 TCP 端口。
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	// 加载服务端证书和私钥。
	// server.pem 给客户端验证服务端身份，server.key 是服务端私钥。
	creds, err := credentials.NewServerTLSFromFile("tls/server.pem", "tls/server.key")
	if err != nil {
		log.Fatalf("Failed to generate credentials %v", err)
	}

	// 创建 gRPC 服务端：
	// - grpc.Creds(creds)：启用 TLS。
	// - grpc.UnaryInterceptor(authInterceptor)：给所有一元 RPC 加 token 校验。
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(authInterceptor),
	)
	pb.RegisterSimpleServer(grpcServer, &simpleService{})

	log.Println(address + " net.Listing with TLS and token...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}

// authInterceptor 是一元拦截器。
// 它会在真正的业务方法 Route 执行前运行，适合做认证、日志、限流等通用逻辑。
func authInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if err := checkToken(ctx); err != nil {
		// 返回错误后，handler 不会执行，客户端会收到 Unauthenticated。
		return nil, err
	}
	// token 校验通过，继续调用真正的业务处理函数。
	return handler(ctx, req)
}

// checkToken 从 metadata 中取出客户端提交的 app_id 和 app_secret。
func checkToken(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "missing token metadata")
	}
	if first(md.Get("app_id")) != "grpc_token" || first(md.Get("app_secret")) != "123456" {
		return status.Error(codes.Unauthenticated, "invalid token")
	}
	return nil
}

// first 取 metadata 某个 key 的第一个值。
// metadata 的值类型是 []string，因为同一个 key 可能出现多次。
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
