package server

import (
	"gin-demo/internal/config"
	"gin-demo/internal/controller"
	"gin-demo/internal/middleware"
	"gin-demo/internal/service"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg    config.Config
	engine *gin.Engine
}

func New(cfg config.Config, authSvc *service.AuthService, postSvc *service.PostService) *Server {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	authController := controller.NewAuthController(authSvc)
	postController := controller.NewPostController(postSvc)

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")
	api.POST("/auth/register", authController.Register)
	api.POST("/auth/login", authController.Login)

	posts := api.Group("/posts")
	posts.GET("", postController.List)
	posts.GET("/:id", postController.Get)

	protected := posts.Group("")
	protected.Use(middleware.Auth(authSvc))
	protected.POST("", postController.Create)
	protected.PUT("/:id", postController.Update)
	protected.DELETE("/:id", postController.Delete)

	return &Server{cfg: cfg, engine: engine}
}

func (s *Server) Run() error {
	return s.engine.Run(s.cfg.HTTPAddr)
}
