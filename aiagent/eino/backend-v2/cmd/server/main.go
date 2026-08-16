package main

import (
	"log"
	"net/http"

	"github.com/mmzhang/aiagent-eino-v2-backend/internal/agent"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/httpapi"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/httpapi/handler"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/modelregistry"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	registry, err := modelregistry.New(cfg.Models.Providers)
	if err != nil {
		log.Fatal(err)
	}
	service := agent.NewService(registry, cfg.Models.Default)
	router := httpapi.NewRouter(cfg.Server, handler.NewChatHandler(service))
	log.Printf("Eino v2 server listening on %s", cfg.Server.Address)
	if err := http.ListenAndServe(cfg.Server.Address, router); err != nil {
		log.Fatal(err)
	}
}
