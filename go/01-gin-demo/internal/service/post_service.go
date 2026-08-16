package service

import (
	"context"

	"gin-demo/internal/model"
	"gin-demo/internal/repository"
)

type PostService struct {
	posts *repository.PostRepository
}

func NewPostService(posts *repository.PostRepository) *PostService {
	return &PostService{posts: posts}
}

func (s *PostService) Create(ctx context.Context, userID int64, req model.CreatePostRequest) (*model.Post, error) {
	return s.posts.Create(ctx, &model.Post{
		Title:   req.Title,
		Content: req.Content,
		UserID:  userID,
	})
}

func (s *PostService) List(ctx context.Context) ([]model.Post, error) {
	return s.posts.List(ctx)
}

func (s *PostService) FindByID(ctx context.Context, id int64) (*model.Post, error) {
	return s.posts.FindByID(ctx, id)
}

func (s *PostService) Update(ctx context.Context, userID, id int64, req model.UpdatePostRequest) (*model.Post, error) {
	return s.posts.Update(ctx, &model.Post{
		ID:      id,
		Title:   req.Title,
		Content: req.Content,
		UserID:  userID,
	})
}

func (s *PostService) Delete(ctx context.Context, userID, id int64) error {
	return s.posts.Delete(ctx, id, userID)
}
