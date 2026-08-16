package main

import (
	"log"
	"net/http"

	"gin-demo/user-service/internal/config"
	"gin-demo/user-service/internal/kratosx"
	"gin-demo/user-service/internal/ratelimit"
	"gin-demo/user-service/internal/repository"
	"gin-demo/user-service/internal/user"
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
	authLimiter := ratelimit.NewPerIP(cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst)
	userHandler := user.NewHandler(userService, authLimiter)

	mux := http.NewServeMux()
	userHandler.RegisterRoutes(mux)

	app := kratosx.NewHTTPApp("blog.user", cfg.HTTPAddr, mux)
	if err := app.Run(); err != nil {
		log.Fatalf("run user service: %v", err)
	}
}
