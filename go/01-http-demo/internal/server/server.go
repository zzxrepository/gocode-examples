package server

import (
	"net/http"
	"time"

	"gin-demo-v1/internal/config"
	"gin-demo-v1/internal/controller"
	"gin-demo-v1/internal/httpapi"
	"gin-demo-v1/internal/middleware"
	"gin-demo-v1/internal/service"
)

type Server struct{ httpServer *http.Server }

func New(cfg config.Config, authSvc *service.AuthService, postSvc *service.PostService) *Server {
	authController := controller.NewAuthController(authSvc)
	postController := controller.NewPostController(postSvc)

	// ServeMux 是标准库路由器。Go 1.22+ 的 pattern 同时表达 HTTP 方法、路径
	// 和路径变量；例如 {id} 通过 r.PathValue("id") 读取。
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/auth/register", authController.Register)
	mux.HandleFunc("POST /api/v1/auth/login", authController.Login)
	mux.HandleFunc("GET /api/v1/posts", postController.List)
	mux.HandleFunc("GET /api/v1/posts/{id}", postController.Get)

	protected := middleware.Auth(authSvc)
	mux.Handle("POST /api/v1/posts", protected(http.HandlerFunc(postController.Create)))
	mux.Handle("PUT /api/v1/posts/{id}", protected(http.HandlerFunc(postController.Update)))
	mux.Handle("DELETE /api/v1/posts/{id}", protected(http.HandlerFunc(postController.Delete)))

	// Chain 由右向左包裹：Logger(Recovery(mux))。
	handler := middleware.Chain(mux, middleware.Logger, middleware.Recovery)
	return &Server{httpServer: &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}}
}

func (s *Server) Run() error { return s.httpServer.ListenAndServe() }
