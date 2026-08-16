package main

import (
	// context.Context 会随着每次 RPC 调用一起传入。
	// 它可以携带超时、取消信号、metadata 等信息。
	"context"

	// log 用来打印服务启动和错误信息。
	"log"

	// net 用来监听 TCP 端口。
	// gRPC 底层也是跑在网络连接上的，这里先创建一个 TCP listener。
	"net"

	// pb 是 generated protobuf 的包别名。
	// 这个包由 proto/simple.proto 生成，里面包含：
	// - SimpleRequest / SimpleResponse 消息类型
	// - SimpleServer 服务端接口
	// - RegisterSimpleServer 注册函数
	pb "go-grpc-simple-rpc/proto"

	// grpc 是 gRPC-Go 的核心包。
	// 服务端用 grpc.NewServer() 创建 gRPC server。
	"google.golang.org/grpc"
)

const (
	// address 是服务端监听地址。
	// ":8000" 表示监听本机所有网卡的 8000 端口。
	// 客户端可以用 "localhost:8000" 连接它。
	address = ":8000"

	// network 是网络协议，这里使用 TCP。
	// gRPC 通常运行在 HTTP/2 + TCP 之上。
	network = "tcp"
)

// simpleService 是我们自己定义的服务实现。
//
// proto 文件只定义了服务“长什么样”，不会自动生成业务逻辑。
// 我们需要创建一个结构体，并实现生成代码里要求的 SimpleServer 接口。
type simpleService struct {
	// 嵌入 UnimplementedSimpleServer 是新版 gRPC-Go 推荐写法。
	// 作用：
	// 1. 如果以后 proto 里新增 RPC 方法，旧代码仍能编译。
	// 2. 未实现的方法会自动返回 Unimplemented 错误。
	pb.UnimplementedSimpleServer
}

// Route 实现 proto 中定义的 rpc Route(SimpleRequest) returns (SimpleResponse)。
//
// 参数说明：
// - ctx：本次 RPC 调用的上下文，可以检查客户端是否取消、是否超时等。
// - req：客户端发来的请求消息。
//
// 返回值说明：
// - *pb.SimpleResponse：返回给客户端的响应消息。
// - error：如果业务处理失败，可以返回错误；成功时返回 nil。
func (s *simpleService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	// req.GetData() 是生成代码提供的安全 getter。
	// 相比直接访问 req.Data，它能更好地处理 req 为 nil 或字段未设置的情况。
	return &pb.SimpleResponse{
		Code:  200,
		Value: "hello " + req.GetData(),
	}, nil
}

func main() {
	// 1. 先监听 TCP 端口。
	//    如果端口已经被占用，net.Listen 会返回错误。
	listener, err := net.Listen(network, address)
	if err != nil {
		// Fatalf 会打印错误并退出程序。
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(address + " net.Listing...")

	// 2. 创建一个 gRPC 服务端实例。
	//    这里没有配置 TLS、拦截器、认证等高级能力，只演示最基础的 Simple RPC。
	grpcServer := grpc.NewServer()

	// 3. 把我们的业务实现注册到 gRPC 服务端。
	//    注册后，客户端调用 Simple.Route 时，gRPC 框架会转发到 simpleService.Route。
	pb.RegisterSimpleServer(grpcServer, &simpleService{})

	// 4. 启动服务端。
	//    Serve 会阻塞当前 goroutine，一直等待客户端请求。
	//    直到进程退出、listener 关闭，或者 grpcServer.Stop/GracefulStop 被调用。
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}
