package main

import (
	"log"

	postapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/api"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/config"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/grpcapi"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/post"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/repository"

	kratos "github.com/go-kratos/kratos/v2"
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := repository.NewMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	postRepo := post.NewRepository(db)
	postService := post.NewService(postRepo)
	server := kgrpc.NewServer(kgrpc.Address(cfg.HTTPAddr))
	postapi.RegisterPostServer(server, grpcapi.New(postService))
	app := kratos.New(kratos.Name("blog.post"), kratos.Server(server))
	if err := app.Run(); err != nil {
		log.Fatalf("run post service: %v", err)
	}
}
