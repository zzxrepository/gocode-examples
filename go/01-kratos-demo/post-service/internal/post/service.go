package post

import (
	"context"

	"gin-demo/post-service/internal/model"
)

type Service struct {
	posts *Repository
}

func NewService(posts *Repository) *Service {
	return &Service{posts: posts}
}

func (s *Service) Create(ctx context.Context, userID int64, req model.CreatePostRequest) (*model.Post, error) {
	return s.posts.Create(ctx, &model.Post{
		Title:   req.Title,
		Content: req.Content,
		UserID:  userID,
	})
}

func (s *Service) List(ctx context.Context) ([]model.Post, error) {
	return s.posts.List(ctx)
}

func (s *Service) FindByID(ctx context.Context, id int64) (*model.Post, error) {
	return s.posts.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, userID, id int64, req model.UpdatePostRequest) (*model.Post, error) {
	return s.posts.Update(ctx, &model.Post{
		ID:      id,
		Title:   req.Title,
		Content: req.Content,
		UserID:  userID,
	})
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	return s.posts.Delete(ctx, id, userID)
}
