package main

import (
	// io.EOF 表示客户端已经关闭发送方向。
	"io"
	"log"
	"net"
	// strconv.Itoa 用来给回答编号。
	"strconv"

	pb "go-grpc-bidirectional-stream-rpc/proto"

	"google.golang.org/grpc"
)

const (
	address = ":8000"
	network = "tcp"
)

type streamService struct {
	pb.UnimplementedStreamServer
}

// Conversations 实现双向流式 RPC。
//
// srv 同时具备 Recv 和 Send 能力：
// - Recv：接收客户端发来的问题。
// - Send：向客户端发送回答。
func (s *streamService) Conversations(srv pb.Stream_ConversationsServer) error {
	// n 用来记录当前是第几个问题。
	n := 1
	for {
		// 读取客户端发来的下一条消息。
		req, err := srv.Recv()
		if err == io.EOF {
			// 客户端关闭发送流后，服务端返回 nil 结束 RPC。
			return nil
		}
		if err != nil {
			return err
		}

		// 收到一个问题后，立即发送一个回答。
		// 这是一问一答模型；双向流式也可以改成两个 goroutine 并发读写。
		if err := srv.Send(&pb.StreamResponse{
			Answer: "from stream server answer: the " + strconv.Itoa(n) + " question is " + req.GetQuestion(),
		}); err != nil {
			return err
		}

		// 服务端打印收到的问题，方便观察双向通信。
		log.Printf("from stream client question: %s", req.GetQuestion())
		n++
	}
}

func main() {
	// 监听 8000 端口并启动 gRPC 服务。
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(address + " net.Listing...")

	grpcServer := grpc.NewServer()
	pb.RegisterStreamServer(grpcServer, &streamService{})

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
