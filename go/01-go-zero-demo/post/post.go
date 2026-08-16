package main

import (
	"flag"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/config"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/server"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/svc"
	pb_postpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/post.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb_postpb.RegisterPostServer(grpcServer, server.NewPostServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
