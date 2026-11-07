package main

import (
	"log"

	"http-demo/internal/config"
	"http-demo/internal/infra/mysql"
	"http-demo/internal/infra/redis"
	"http-demo/internal/repository"
	"http-demo/internal/server"
	"http-demo/internal/service"
)

func main() {
	cfg := config.Load()
	db, err := mysql.NewClient(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()
	rdb, err := redis.NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer rdb.Close()

	userRepo := repository.NewUserRepository(db)
	postRepo := repository.NewPostRepository(db)
	sessionRepo := repository.NewSessionRepository(rdb)
	authSvc := service.NewAuthService(userRepo, sessionRepo, cfg.JWTSecret, cfg.JWTExpire)
	postSvc := service.NewPostService(postRepo)

	log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
	if err := server.New(cfg, authSvc, postSvc).Run(); err != nil {
		log.Fatal(err)
	}
}
