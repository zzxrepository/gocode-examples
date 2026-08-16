package main

import (
	"context"
	"log"
	"time"

	// auth.Token 负责给每次 RPC 添加 authorization metadata。
	"go-grpc-middleware-example/internal/auth"
	pb "go-grpc-middleware-example/proto"

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

	// bearer token 会被服务端 Auth 拦截器校验。
	token := auth.Token{Value: "bearer grpc.auth.token"}

	// 建立同时带 TLS 和 PerRPCCredentials 的连接。
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

	// 给本次 RPC 设置 5 秒超时。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 调用会经过服务端拦截器链，然后进入 Route。
	res, err := grpcClient.Route(ctx, &pb.SimpleRequest{Data: "grpc"})
	if err != nil {
		log.Fatalf("Call Route err: %v", err)
	}
	// 正常输出：code:200 value:"hello grpc"。
	log.Println(res)
}
