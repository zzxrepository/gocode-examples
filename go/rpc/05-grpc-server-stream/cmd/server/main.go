package main

import (
	// log 打印服务启动信息和错误。
	"log"
	// net 用来创建 TCP 监听端口。
	"net"
	// strconv.Itoa 把数字转成字符串，方便拼接响应内容。
	"strconv"

	// pb 是 proto 生成的包，里面有请求/响应类型和服务注册函数。
	pb "go-grpc-server-stream-rpc/proto"

	// grpc 是 gRPC-Go 的核心服务端包。
	"google.golang.org/grpc"
)

const (
	// ":8000" 表示监听所有网卡的 8000 端口。
	address = ":8000"
	// gRPC 底层基于 HTTP/2，HTTP/2 又跑在 TCP 连接上。
	network = "tcp"
)

// streamService 是服务端业务实现。
// 它需要实现 proto 中定义的 StreamServerServer 接口。
type streamService struct {
	// 嵌入未实现服务是新版 gRPC-Go 的推荐写法，方便以后 proto 新增方法时保持兼容。
	pb.UnimplementedStreamServerServer
}

// ListValue 实现服务端流式 RPC。
//
// 和简单 RPC 不同，这里的返回值不是 *StreamResponse。
// 服务端要通过 srv.Send(...) 多次把响应发给客户端。
func (s *streamService) ListValue(req *pb.SimpleRequest, srv pb.StreamServer_ListValueServer) error {
	// 模拟服务端连续返回 5 条数据。
	for n := 0; n < 5; n++ {
		// 每调用一次 Send，客户端就可以通过 Recv 收到一条 StreamResponse。
		if err := srv.Send(&pb.StreamResponse{StreamValue: req.GetData() + strconv.Itoa(n)}); err != nil {
			// 如果客户端断开连接或网络异常，Send 会返回错误。
			return err
		}
	}
	// 返回 nil 表示服务端发送完毕，客户端最终会读到 io.EOF。
	return nil
}

func main() {
	// 创建 TCP listener，等待客户端连接。
	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(address + " net.Listing...")

	// 创建 gRPC server 并注册当前服务实现。
	grpcServer := grpc.NewServer()
	pb.RegisterStreamServerServer(grpcServer, &streamService{})

	// Serve 会阻塞运行，直到服务关闭或出错。
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
