package main

import (
	// io.EOF 用来判断客户端是否已经发送完全部消息。
	"io"
	"log"
	"net"

	pb "go-grpc-client-stream-rpc/proto"

	"google.golang.org/grpc"
)

const (
	address = ":8000"
	network = "tcp"
)

// streamClientService 实现 proto 中的 StreamClientServer 接口。
type streamClientService struct {
	pb.UnimplementedStreamClientServer
}

// RouteList 实现客户端流式 RPC。
//
// 参数 srv 同时负责接收客户端消息和发送最终响应：
// - srv.Recv()：读取客户端发来的下一条 StreamRequest。
// - srv.SendAndClose()：发送最终 SimpleResponse，并关闭服务端响应。
func (s *streamClientService) RouteList(srv pb.StreamClient_RouteListServer) error {
	for {
		// 不断读取客户端发来的消息。
		req, err := srv.Recv()
		if err == io.EOF {
			// io.EOF 表示客户端已经 CloseAndRecv / CloseSend，不再发送数据。
			// 服务端此时返回一个最终响应。
			return srv.SendAndClose(&pb.SimpleResponse{
				Code:  200,
				Value: "ok",
			})
		}
		if err != nil {
			return err
		}
		// 这里模拟服务端处理每一条上传数据。
		log.Println(req.GetStreamData())
	}
}

func main() {
	// 创建 TCP listener。
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(address + " net.Listing...")

	// 创建并注册 gRPC 服务。
	grpcServer := grpc.NewServer()
	pb.RegisterStreamClientServer(grpcServer, &streamClientService{})

	// 阻塞运行服务端。
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
