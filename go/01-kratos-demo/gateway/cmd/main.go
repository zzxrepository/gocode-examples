package main

import (
	"log"
	"net/http"

	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/gateway/internal/config"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/gateway/internal/gateway"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/gateway/internal/kratosx"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/gateway/internal/ratelimit"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	userClient, err := gateway.NewUserClient(cfg.UserServiceURL)
	if err != nil {
		log.Fatalf("connect user rpc: %v", err)
	}
	postClient, err := gateway.NewPostClient(cfg.PostServiceURL)
	if err != nil {
		log.Fatalf("connect post rpc: %v", err)
	}
	gatewayHandler := gateway.NewHandler(userClient, postClient)

	mux := http.NewServeMux()
	gatewayHandler.RegisterRoutes(mux)

	limitedMux := ratelimit.NewPerIP(cfg.RateLimitRPS, cfg.RateLimitBurst).Middleware(mux)
	app := kratosx.NewHTTPApp("blog.gateway", cfg.HTTPAddr, limitedMux)
	if err := app.Run(); err != nil {
		log.Fatalf("run gateway: %v", err)
	}
}
