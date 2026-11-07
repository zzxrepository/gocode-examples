package server

import (
	"net/http"
	"time"

	"http-demo/internal/config"
	"http-demo/internal/handler"
	"http-demo/internal/httpapi"
	"http-demo/internal/middleware"
	"http-demo/internal/service"
)

type Server struct {
	httpServer *http.Server
}

func New(cfg config.Config, authSvc *service.AuthService, postSvc *service.PostService) *Server {
	authHandler := handler.NewAuthHandler(authSvc)
	postHandler := handler.NewPostHandler(postSvc)

	// ServeMux 是标准库路由器。Go 1.22+ 的 pattern 同时表达 HTTP 方法、路径
	// 和路径变量；例如 {id} 通过 r.PathValue("id") 读取。
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("GET /api/v1/posts", postHandler.List)
	mux.HandleFunc("GET /api/v1/posts/{id}", postHandler.Get)

	protected := middleware.Auth(authSvc)
	mux.Handle("POST /api/v1/posts", protected(http.HandlerFunc(postHandler.Create)))
	mux.Handle("PUT /api/v1/posts/{id}", protected(http.HandlerFunc(postHandler.Update)))
	mux.Handle("DELETE /api/v1/posts/{id}", protected(http.HandlerFunc(postHandler.Delete)))

	// Chain 由右向左包裹：Logger(Recovery(mux))。
	httpHandler := middleware.Chain(mux, middleware.Logger, middleware.Recovery)
	return &Server{httpServer: &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}
