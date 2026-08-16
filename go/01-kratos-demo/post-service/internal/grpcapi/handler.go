package grpcapi

import (
	"context"

	postapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/api"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/model"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/post"
)

type Handler struct {
	postapi.UnimplementedPostServer
	service *post.Service
}

func New(service *post.Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(ctx context.Context, _ *postapi.ListRequest) (*postapi.PostListReply, error) {
	posts, err := h.service.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*postapi.PostReply, 0, len(posts))
	for _, item := range posts {
		result = append(result, toReply(&item))
	}
	return &postapi.PostListReply{Posts: result}, nil
}

func (h *Handler) Get(ctx context.Context, req *postapi.PostIDRequest) (*postapi.PostReply, error) {
	item, err := h.service.FindByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return toReply(item), nil
}

func (h *Handler) Create(ctx context.Context, req *postapi.CreateRequest) (*postapi.PostReply, error) {
	item, err := h.service.Create(ctx, req.UserId, model.CreatePostRequest{Title: req.Title, Content: req.Content})
	if err != nil {
		return nil, err
	}
	return toReply(item), nil
}

func (h *Handler) Update(ctx context.Context, req *postapi.UpdateRequest) (*postapi.PostReply, error) {
	item, err := h.service.Update(ctx, req.UserId, req.Id, model.UpdatePostRequest{Title: req.Title, Content: req.Content})
	if err != nil {
		return nil, err
	}
	return toReply(item), nil
}

func (h *Handler) Delete(ctx context.Context, req *postapi.DeleteRequest) (*postapi.Empty, error) {
	if err := h.service.Delete(ctx, req.UserId, req.Id); err != nil {
		return nil, err
	}
	return &postapi.Empty{}, nil
}

func toReply(item *model.Post) *postapi.PostReply {
	return &postapi.PostReply{Id: item.ID, UserId: item.UserID, Title: item.Title, Content: item.Content}
}
