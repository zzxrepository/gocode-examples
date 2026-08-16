package main

import (
	// context 用来给 RPC 调用设置超时/取消。
	"context"
	// io.EOF 表示服务端流已经结束。
	"io"
	// log 用来打印接收到的数据。
	"log"
	// time 用来设置 5 秒调用超时。
	"time"

	// pb 是 proto 生成的客户端类型所在包。
	pb "go-grpc-server-stream-rpc/proto"

	"google.golang.org/grpc"
	// 本地示例不启用 TLS，所以使用 insecure 凭证。
	"google.golang.org/grpc/credentials/insecure"
)

// 服务端监听 ":8000"，客户端连接时写 "localhost:8000"。
const address = "localhost:8000"

func main() {
	// 建立到 gRPC 服务端的连接。
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc.NewClient err: %v", err)
	}
	defer conn.Close()

	// 创建 StreamServer 服务的客户端。
	grpcClient := pb.NewStreamServerClient(conn)

	// 本次 RPC 最多等待 5 秒。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 发起请求。因为这是服务端流式 RPC，所以返回的是 stream，而不是单个响应。
	stream, err := grpcClient.ListValue(ctx, &pb.SimpleRequest{Data: "stream server grpc "})
	if err != nil {
		log.Fatalf("Call ListValue err: %v", err)
	}

	// 循环从响应流中读取数据。
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			// io.EOF 说明服务端已经发送完全部消息，循环可以结束。
			break
		}
		if err != nil {
			log.Fatalf("ListValue get stream err: %v", err)
		}
		// 打印每一条服务端返回的数据。
		log.Println(res.GetStreamValue())
	}
}
