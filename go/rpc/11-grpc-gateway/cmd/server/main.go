package main

import (
	"context"
	"log"
	"net"
	// net/http 用来启动 HTTP gateway。
	"net/http"
	// strings.EqualFold 用来忽略大小写匹配 Authorization 请求头。
	"strings"

	"go-grpc-gateway-example/internal/interceptor"
	pb "go-grpc-gateway-example/proto"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	// gRPC 服务监听 8000。
	grpcAddress = "localhost:8000"
	// HTTP gateway 监听 8080。
	// 这里和 gRPC 拆成两个端口，学习时更直观。
	httpAddress = "localhost:8080"
	network     = "tcp"
)

type simpleService struct {
	pb.UnimplementedSimpleServer
}

// Route 是真正的业务方法。
// 无论请求来自 gRPC 客户端还是 HTTP gateway，最终都会执行到这里。
func (s *simpleService) Route(ctx context.Context, req *pb.InnerMessage) (*pb.OuterMessage, error) {
	return &pb.OuterMessage{
		// "hello" 满足 proto 里 important_string 的校验规则。
		ImportantString: "hello",
		Inner:           req,
	}, nil
}

func main() {
	// 先创建 gRPC listener。
	listener, err := net.Listen(network, grpcAddress)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}

	// gRPC 服务端启用 TLS。
	serverCreds, err := credentials.NewServerTLSFromFile("tls/server.pem", "tls/server.key")
	if err != nil {
		log.Fatalf("Failed to generate credentials %v", err)
	}

	// gRPC 服务端同时使用校验和认证拦截器。
	grpcServer := grpc.NewServer(
		grpc.Creds(serverCreds),
		grpc.ChainUnaryInterceptor(interceptor.Validate, interceptor.Auth),
	)
	pb.RegisterSimpleServer(grpcServer, &simpleService{})

	// gRPC 服务需要先启动起来，因为 HTTP gateway 会把 HTTP 请求转发到它。
	// 放到 goroutine 中运行，主 goroutine 继续启动 HTTP gateway。
	go func() {
		log.Println(grpcAddress + " gRPC listening with TLS and token...")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("grpcServer.Serve err: %v", err)
		}
	}()

	// 启动 HTTP gateway。这个函数会阻塞，直到 HTTP 服务停止或出错。
	if err := serveGateway(); err != nil {
		log.Fatalf("gateway err: %v", err)
	}
}

// serveGateway 创建 HTTP 网关。
// 它接收 HTTP JSON 请求，然后调用本机的 gRPC 服务。
func serveGateway() error {
	// gateway 作为“gRPC 客户端”连接 grpcAddress，所以也需要客户端 TLS 凭证。
	clientCreds, err := credentials.NewClientTLSFromFile("tls/server.pem", "go-grpc-example")
	if err != nil {
		return err
	}

	// NewServeMux 是 grpc-gateway 的 HTTP 路由器。
	mux := runtime.NewServeMux(
		// 默认情况下，gateway 不会把所有 HTTP header 都转成 gRPC metadata。
		// 这里显式允许 Authorization 透传，服务端 Auth 拦截器才能拿到 token。
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			if strings.EqualFold(key, "Authorization") {
				return key, true
			}
			return runtime.DefaultHeaderMatcher(key)
		}),
	)

	// 注册由 protoc-gen-grpc-gateway 生成的 HTTP handler。
	// 它知道 proto 中 google.api.http 注解描述的路由规则。
	opts := []grpc.DialOption{grpc.WithTransportCredentials(clientCreds)}
	if err := pb.RegisterSimpleHandlerFromEndpoint(context.Background(), mux, grpcAddress, opts); err != nil {
		return err
	}

	// 启动 HTTP 服务。示例里 HTTP gateway 自身不启用 TLS，
	// 但 gateway 到 gRPC 服务的内部连接是 TLS。
	log.Println(httpAddress + " HTTP gateway listening...")
	return http.ListenAndServe(httpAddress, mux)
}
