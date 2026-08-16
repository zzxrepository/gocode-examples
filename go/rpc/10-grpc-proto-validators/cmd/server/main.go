package main

import (
	"context"
	"log"
	"net"

	// interceptor.Validate 负责调用生成的 Validate() 方法。
	// interceptor.Auth 负责 token 认证。
	"go-grpc-proto-validators/internal/interceptor"
	pb "go-grpc-proto-validators/proto"

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

// Route 的 req 是 InnerMessage。
// 在进入这里之前，Validate 拦截器已经检查 some_integer 和 some_float 是否合法。
func (s *simpleService) Route(ctx context.Context, req *pb.InnerMessage) (*pb.OuterMessage, error) {
	return &pb.OuterMessage{
		// "hello" 满足 proto 里的正则 ^[a-z]{2,5}$。
		ImportantString: "hello",
		Inner:           req,
	}, nil
}

func main() {
	// 启动带 TLS 的 gRPC 服务端。
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	// 加载服务端 TLS 证书。
	creds, err := credentials.NewServerTLSFromFile("tls/server.pem", "tls/server.key")
	if err != nil {
		log.Fatalf("Failed to generate credentials %v", err)
	}

	// 拦截器顺序：
	// 1. Validate：先确认请求字段合法。
	// 2. Auth：再确认调用方有权限。
	// 3. Route：最后进入业务方法。
	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(interceptor.Validate, interceptor.Auth),
	)
	pb.RegisterSimpleServer(grpcServer, &simpleService{})

	log.Println(address + " net.Listing with TLS, token and validators...")
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
