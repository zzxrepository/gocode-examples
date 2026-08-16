package main

import (
	// context 用来控制连接和 RPC 调用的生命周期。
	// 例如设置超时，防止客户端无限等待。
	"context"

	// log 用来打印调用结果和错误信息。
	"log"

	// time 用来设置超时时间。
	"time"

	// pb 是由 proto/simple.proto 生成的 Go 包。
	// 客户端会使用里面的 NewSimpleClient 创建 gRPC 客户端。
	pb "go-grpc-simple-rpc/proto"

	// grpc 是 gRPC-Go 的核心包。
	// 客户端用 grpc.DialContext 建立到服务端的连接。
	"google.golang.org/grpc"

	// insecure.NewCredentials 表示“不使用 TLS 加密”的连接凭证。
	// 这是本地入门教程常用写法，生产环境通常应该使用 TLS。
	"google.golang.org/grpc/credentials/insecure"
)

// address 是服务端地址。
// 服务端监听 ":8000"，客户端连接时通常写成 "localhost:8000"。
const address = "localhost:8000"

func main() {
	// 1. 为“建立连接”设置 5 秒超时。
	//    如果服务端没启动，客户端不会一直卡住，而是在 5 秒后失败。
	connCtx, connCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer connCancel()

	// 2. 建立到 gRPC 服务端的连接。
	//
	// grpc.DialContext 只负责创建连接对象；加上 grpc.WithBlock() 后，
	// 它会等待连接真正建立成功或 connCtx 超时。
	conn, err := grpc.DialContext(
		connCtx,
		address,
		// 本示例没有配置 TLS，所以使用 insecure 凭证。
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// WithBlock 让 DialContext 阻塞等待连接完成。
		// 对入门示例很友好，因为服务端没启动时能立刻看到连接错误。
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("grpc.DialContext err: %v", err)
	}
	// 程序退出前关闭连接，释放底层网络资源。
	defer conn.Close()

	// 3. 创建 Simple 服务的客户端。
	//    NewSimpleClient 是 protoc 根据 service Simple 自动生成的函数。
	grpcClient := pb.NewSimpleClient(conn)

	// 4. 为“本次 RPC 调用”设置 5 秒超时。
	//    注意：连接超时和调用超时是两件事，所以这里单独创建 callCtx。
	callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer callCancel()

	// 5. 构造请求并调用服务端的 Route 方法。
	//
	// 这就是 Simple RPC：
	// - 客户端发送一个 SimpleRequest{Data: "grpc"}
	// - 服务端返回一个 SimpleResponse
	res, err := grpcClient.Route(callCtx, &pb.SimpleRequest{Data: "grpc"})
	if err != nil {
		log.Fatalf("Call Route err: %v", err)
	}

	// 6. 打印服务端响应。
	//    正常情况下会看到：code:200 value:"hello grpc"
	log.Println(res)
}
