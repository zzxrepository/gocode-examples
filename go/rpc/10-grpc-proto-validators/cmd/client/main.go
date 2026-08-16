package main

import (
	"context"
	"log"
	"time"

	// auth.Token 用来发送 bearer token。
	"go-grpc-proto-validators/internal/auth"
	pb "go-grpc-proto-validators/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const address = "localhost:8000"

func main() {
	// 创建 TLS 客户端凭证。
	creds, err := credentials.NewClientTLSFromFile("tls/server.pem", "go-grpc-example")
	if err != nil {
		log.Fatalf("Failed to create TLS credentials %v", err)
	}

	// 正确 token，服务端 Auth 拦截器会接受。
	token := auth.Token{Value: "bearer grpc.auth.token"}

	// 建立连接时同时配置 TLS 和每次 RPC 的 token。
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

	// 本次调用最多等待 5 秒。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 构造满足校验规则的请求：
	// - SomeInteger=99，满足 0 < x < 100。
	// - SomeFloat=0.5，满足 0 <= x <= 1。
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
