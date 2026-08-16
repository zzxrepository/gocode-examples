package httpapi

import (
	"net/http"

	"github.com/mmzhang/aiagent-eino-backend/internal/config"
	"github.com/mmzhang/aiagent-eino-backend/internal/httpapi/handler"
	"github.com/mmzhang/aiagent-eino-backend/internal/httpapi/response"
)

func NewRouter(serverConfig config.ServerConfig, chatHandler *handler.ChatHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", chatHandler.Health)
	mux.HandleFunc("GET /api/models", chatHandler.Models)
	mux.HandleFunc("POST /api/chat/stream", chatHandler.StreamChat)
	return withCORS(serverConfig.CORSOrigins, response.WithTraceID(mux))
}

func withCORS(origins []string, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Trace-ID")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
