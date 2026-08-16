package main

import (
	"log"

	userapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/api"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/config"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/grpcapi"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/repository"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/user"

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

	rdb, err := repository.NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer rdb.Close()

	userRepo := user.NewRepository(db)
	sessionRepo := user.NewSessionRepository(rdb)
	userService := user.NewService(userRepo, sessionRepo, cfg.JWTSecret, cfg.JWTExpire)
	server := kgrpc.NewServer(kgrpc.Address(cfg.HTTPAddr))
	userapi.RegisterUserServer(server, grpcapi.New(userService))
	app := kratos.New(kratos.Name("blog.user"), kratos.Server(server))
	if err := app.Run(); err != nil {
		log.Fatalf("run user service: %v", err)
	}
}
