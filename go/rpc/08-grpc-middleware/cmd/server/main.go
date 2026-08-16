package main

import (
	"context"
	"log"
	"net"

	// interceptor 包里放了认证、日志、恢复等通用拦截器。
	"go-grpc-middleware-example/internal/interceptor"
	pb "go-grpc-middleware-example/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	address = ":8000"
	network = "tcp"
)

type simpleService struct {
	pb.UnimplementedSimpleServer
}

// Route 是真正的业务处理方法。
// 认证、日志、panic 恢复都在拦截器中完成，所以这里保持业务逻辑简单。
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

	// 加载 TLS 证书。middleware 示例同时演示 TLS + token + 拦截器。
	creds, err := credentials.NewServerTLSFromFile("tls/server.pem", "tls/server.key")
	if err != nil {
		log.Fatalf("Failed to generate credentials %v", err)
	}

	// 初始化日志器。Logging 拦截器会使用它写入 log/debug.log。
	logger, err := interceptor.NewLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger %v", err)
	}
	defer logger.Sync()

	// 创建 gRPC server，并串联多个一元拦截器。
	//
	// 执行顺序：
	// 1. Recovery：最外层，兜住后续逻辑的 panic。
	// 2. Logging：记录方法、状态码、耗时。
	// 3. Auth：校验 token。
	// 4. Route：真正的业务方法。
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			interceptor.Recovery,
			interceptor.Logging(logger),
			interceptor.Auth,
		),
	)
	// 注册服务实现。
	pb.RegisterSimpleServer(grpcServer, &simpleService{})

	log.Println(address + " net.Listing with TLS, token and interceptors...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
