// Package routes assembles HTTP endpoints from handlers and middleware.
package routes

import (
	"net/http"

	"github.com/mmzhang/aiagent-eino-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-backend/internal/handler"
	"github.com/mmzhang/aiagent-eino-backend/internal/middleware"
)

func NewRouter(serverConfig config.ServerConfig, chatHandler *handler.ChatHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", chatHandler.Health)
	mux.HandleFunc("GET /api/models", chatHandler.Models)
	mux.HandleFunc("POST /api/chat/stream", chatHandler.StreamChat)
	return middleware.WithCORS(serverConfig.CORSOrigins, middleware.WithTraceID(mux))
}
