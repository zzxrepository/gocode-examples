package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/mmzhang/aiagent-eino-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-backend/internal/httpapi"
	"github.com/mmzhang/aiagent-eino-backend/internal/httpapi/handler"
	"github.com/mmzhang/aiagent-eino-backend/internal/provider"
	"github.com/mmzhang/aiagent-eino-backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	registry, err := provider.NewRegistry(cfg.Models.Providers)
	if err != nil {
		log.Fatal(err)
	}
	chatService := service.NewChatService(registry, cfg.Models.Default)
	router := httpapi.NewRouter(cfg.Server, handler.NewChatHandler(chatService))

	server := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      100 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("AI Agent API listening on http://localhost%s", cfg.Server.Address)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
