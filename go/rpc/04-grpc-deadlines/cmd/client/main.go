package main

import (
	// context.WithTimeout 用来给某次 RPC 设置 deadline。
	"context"
	"log"
	"time"

	pb "go-grpc-deadlines/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const address = "localhost:8000"

func main() {
	// 本地明文连接。deadline 示例关注调用超时，不关注 TLS。
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc.NewClient err: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewSimpleClient(conn)

	// 服务端会处理 4 秒。
	// 第一次只给 2 秒，因此会超时。
	callRoute(grpcClient, 2*time.Second)

	// 第二次给 5 秒，因此能正常拿到响应。
	callRoute(grpcClient, 5*time.Second)
}

// callRoute 调用服务端 Route，并为这次调用设置不同的超时时间。
func callRoute(grpcClient pb.SimpleClient, timeout time.Duration) {
	// WithTimeout 返回一个带超时的 ctx。
	// 超时到达时，ctx 会自动取消，并通知客户端和服务端。
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 把带 deadline 的 ctx 传给 Route。
	res, err := grpcClient.Route(ctx, &pb.SimpleRequest{Data: "grpc"})
	if err != nil {
		// gRPC 错误可以通过 status.Code 取出标准错误码。
		if status.Code(err) == codes.DeadlineExceeded {
			log.Printf("Route timeout after %s", timeout)
			return
		}
		log.Fatalf("Call Route err: %v", err)
	}
	// 调用成功时打印服务端返回内容。
	log.Println(res.GetValue())
}
