package main

import (
	"log"

	"gin-demo-v1/internal/config"
	"gin-demo-v1/internal/repository"
	"gin-demo-v1/internal/server"
	"gin-demo-v1/internal/service"
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

	log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
	if err := server.New(cfg, authSvc, postSvc).Run(); err != nil {
		log.Fatal(err)
	}
}
