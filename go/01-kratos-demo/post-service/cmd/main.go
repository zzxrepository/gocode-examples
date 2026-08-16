package main

import (
	"log"
	"net/http"

	"gin-demo/post-service/internal/config"
	"gin-demo/post-service/internal/kratosx"
	"gin-demo/post-service/internal/post"
	"gin-demo/post-service/internal/repository"
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

	postRepo := post.NewRepository(db)
	postService := post.NewService(postRepo)
	postHandler := post.NewHandler(postService)

	mux := http.NewServeMux()
	postHandler.RegisterRoutes(mux)

	app := kratosx.NewHTTPApp("blog.post", cfg.HTTPAddr, mux)
	if err := app.Run(); err != nil {
		log.Fatalf("run post service: %v", err)
	}
}
