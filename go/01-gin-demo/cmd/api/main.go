package main

import (
	"log"

	"gin-demo/internal/config"
	"gin-demo/internal/repository"
	"gin-demo/internal/server"
	"gin-demo/internal/service"
)

func main() {
	cfg := config.Load()

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

	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	sessionRepo := repository.NewSessionRepository(rdb)

	authSvc := service.NewAuthService(userRepo, sessionRepo, cfg.JWTSecret, cfg.JWTExpire)
	postSvc := service.NewPostService(postRepo)

	srv := server.New(cfg, authSvc, postSvc)
	if err := srv.Run(); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
