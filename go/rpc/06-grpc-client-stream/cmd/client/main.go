package main

import (
	// context 控制本次流式 RPC 的超时。
	"context"
	// io.EOF 可能出现在服务端提前关闭流的情况。
	"io"
	"log"
	// strconv.Itoa 把循环数字转成字符串。
	"strconv"
	"time"

	pb "go-grpc-client-stream-rpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const address = "localhost:8000"

func main() {
	// 建立到服务端的连接。本地示例未开启 TLS，所以使用 insecure。
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc.NewClient err: %v", err)
	}
	defer conn.Close()

	// 创建 StreamClient 服务的客户端。
	streamClient := pb.NewStreamClientClient(conn)

	// 给整个上传流程设置 5 秒超时。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 调用客户端流式 RPC，拿到可写的 stream。
	stream, err := streamClient.RouteList(ctx)
	if err != nil {
		log.Fatalf("Upload list err: %v", err)
	}

	// 连续向服务端发送 5 条消息。
	for n := 0; n < 5; n++ {
		err := stream.Send(&pb.StreamRequest{
			StreamData: "stream client rpc " + strconv.Itoa(n),
		})
		if err == io.EOF {
			// 如果服务端提前 SendAndClose，继续 Send 可能会得到 io.EOF。
			// 这时跳出发送循环即可。
			break
		}
		if err != nil {
			log.Fatalf("stream request err: %v", err)
		}
	}

	// CloseAndRecv 表示：
	// 1. 客户端告诉服务端“我发完了”。
	// 2. 等待服务端返回最终响应。
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("RouteList get response err: %v", err)
	}
	// 正常输出：code:200 value:"ok"。
	log.Println(res)
}
