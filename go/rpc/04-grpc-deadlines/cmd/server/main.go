package main

import (
	// context 用来接收客户端的取消/超时信号。
	"context"
	"log"
	"net"
	// time.After 用来模拟一个耗时 4 秒的业务操作。
	"time"

	pb "go-grpc-deadlines/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	address = ":8000"
	network = "tcp"
)

type simpleService struct {
	pb.UnimplementedSimpleServer
}

// Route 演示服务端如何感知客户端 deadline。
//
// 客户端给本次调用设置超时时间后，这个超时会通过 ctx 传到服务端。
// 服务端应该在耗时操作中监听 ctx.Done()，避免客户端已经放弃后服务端还继续浪费资源。
func (s *simpleService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	select {
	case <-time.After(4 * time.Second):
		// 模拟业务处理耗时 4 秒。
		// 如果客户端给的超时时间超过 4 秒，就能正常拿到这个响应。
		return &pb.SimpleResponse{
			Code:  200,
			Value: "hello " + req.GetData(),
		}, nil
	case <-ctx.Done():
		// 如果客户端超时或主动取消，ctx.Done() 会先收到信号。
		// 这里返回 DeadlineExceeded，方便客户端区分“超时”与普通业务错误。
		return nil, status.Error(codes.DeadlineExceeded, "client deadline exceeded")
	}
}

func main() {
	// 启动普通 gRPC 服务端。deadline 相关逻辑主要在 Route 的 ctx 中体现。
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(address + " net.Listing...")

	grpcServer := grpc.NewServer()
	pb.RegisterSimpleServer(grpcServer, &simpleService{})

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
