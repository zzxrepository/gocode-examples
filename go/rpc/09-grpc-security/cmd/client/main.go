package main

import (
	"context"
	"log"
	"time"

	// auth.Token 会给每次 RPC 自动附加认证 metadata。
	"go-grpc-security/internal/auth"
	pb "go-grpc-security/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const address = "localhost:8000"

func main() {
	// 从服务端证书文件创建客户端 TLS 凭证。
	// 第二个参数 "go-grpc-example" 是证书里用于校验的服务名。
	creds, err := credentials.NewClientTLSFromFile("tls/server.pem", "go-grpc-example")
	if err != nil {
		log.Fatalf("Failed to create TLS credentials %v", err)
	}

	// 构造 token。它会被 WithPerRPCCredentials 自动调用，
	// 最终变成 metadata: app_id=grpc_token, app_secret=123456。
	token := auth.Token{
		AppID:     "grpc_token",
		AppSecret: "123456",
	}

	// 建立带 TLS 和 token 的 gRPC 连接。
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

	// 给业务调用设置超时，避免网络异常时一直等待。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发起受保护的 RPC 调用。
	res, err := grpcClient.Route(ctx, &pb.SimpleRequest{Data: "grpc"})
	if err != nil {
		log.Fatalf("Call Route err: %v", err)
	}
	// 正常输出：code:200 value:"hello grpc"。
	log.Println(res)
}
