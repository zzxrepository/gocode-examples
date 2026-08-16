package main

import (
	"context"
	"log"
	"time"

	// auth.Token 给 gRPC 直连客户端附加 Authorization metadata。
	"go-grpc-gateway-example/internal/auth"
	pb "go-grpc-gateway-example/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const address = "localhost:8000"

func main() {
	// 这个客户端演示“直接用 gRPC 调用”，不是 HTTP gateway 调用。
	// HTTP 调用可以看 README 里的 curl 示例。
	creds, err := credentials.NewClientTLSFromFile("tls/server.pem", "go-grpc-example")
	if err != nil {
		log.Fatalf("Failed to create TLS credentials %v", err)
	}

	// gRPC 直连时 token 通过 PerRPCCredentials 进入 metadata。
	token := auth.Token{Value: "bearer grpc.auth.token"}
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(&token),
	)
	if err != nil {
		log.Fatalf("grpc.NewClient err: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewSimpleClient(conn)

	// 给本次调用设置超时。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 构造满足 validator 规则的请求。
	res, err := grpcClient.Route(ctx, &pb.InnerMessage{
		SomeInteger: 99,
		SomeFloat:   0.5,
	})
	if err != nil {
		log.Fatalf("Call Route err: %v", err)
	}
	// 正常输出 important_string 和 inner 内容。
	log.Println(res)
}
