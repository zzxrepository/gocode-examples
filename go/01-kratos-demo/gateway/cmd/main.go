package main

import (
	"log"
	"net/http"

	"gin-demo/gateway/internal/config"
	"gin-demo/gateway/internal/gateway"
	"gin-demo/gateway/internal/kratosx"
	"gin-demo/gateway/internal/ratelimit"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	userClient := gateway.NewUserClient(cfg.UserServiceURL)
	postClient := gateway.NewPostClient(cfg.PostServiceURL)
	gatewayHandler := gateway.NewHandler(userClient, postClient)

	mux := http.NewServeMux()
	gatewayHandler.RegisterRoutes(mux)

	limitedMux := ratelimit.NewPerIP(cfg.RateLimitRPS, cfg.RateLimitBurst).Middleware(mux)
	app := kratosx.NewHTTPApp("blog.gateway", cfg.HTTPAddr, limitedMux)
	if err := app.Run(); err != nil {
		log.Fatalf("run gateway: %v", err)
	}
}
