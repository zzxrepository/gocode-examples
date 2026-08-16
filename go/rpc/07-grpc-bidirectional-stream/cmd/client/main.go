package main

import (
	// context 设置整个对话 RPC 的超时。
	"context"
	// io.EOF 表示服务端响应流结束。
	"io"
	"log"
	// strconv.Itoa 生成问题编号。
	"strconv"
	"time"

	pb "go-grpc-bidirectional-stream-rpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const address = "localhost:8000"

func main() {
	// 建立本地明文连接。
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("grpc.NewClient err: %v", err)
	}
	defer conn.Close()

	// 创建 Stream 服务客户端。
	streamClient := pb.NewStreamClient(conn)

	// 设置整个双向流调用最多 5 秒。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 打开双向流。返回的 stream 既可以 Send，也可以 Recv。
	stream, err := streamClient.Conversations(ctx)
	if err != nil {
		log.Fatalf("get conversations stream err: %v", err)
	}

	// 这里用简单的一问一答演示：
	// 每轮先 Send 一个问题，再 Recv 一个回答。
	for n := 0; n < 5; n++ {
		if err := stream.Send(&pb.StreamRequest{
			Question: "stream client rpc " + strconv.Itoa(n),
		}); err != nil {
			log.Fatalf("stream request err: %v", err)
		}

		// 读取服务端针对当前问题返回的回答。
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Conversations get stream err: %v", err)
		}
		log.Println(res.GetAnswer())
	}

	// CloseSend 只关闭客户端的“发送方向”。
	// 对双向流来说，关闭发送方向并不等于强制关闭整个连接。
	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Conversations close stream err: %v", err)
	}
}
