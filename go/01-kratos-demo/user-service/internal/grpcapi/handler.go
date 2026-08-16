package grpcapi

import (
	"context"
	"strings"

	userapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/api"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/model"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/user"
)

type Handler struct {
	userapi.UnimplementedUserServer
	service *user.Service
}

func New(service *user.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(ctx context.Context, req *userapi.RegisterRequest) (*userapi.UserReply, error) {
	result, err := h.service.Register(ctx, model.RegisterRequest{Username: req.Username, Password: req.Password})
	if err != nil {
		return nil, err
	}
	return &userapi.UserReply{UserId: result.ID, Username: result.Username}, nil
}

func (h *Handler) Login(ctx context.Context, req *userapi.LoginRequest) (*userapi.TokenReply, error) {
	result, err := h.service.Login(ctx, model.LoginRequest{Username: req.Username, Password: req.Password})
	if err != nil {
		return nil, err
	}
	claims, err := h.service.ValidateToken(ctx, result.Token)
	if err != nil {
		return nil, err
	}
	return &userapi.TokenReply{Token: result.Token, UserId: claims.UserID}, nil
}

func (h *Handler) ValidateToken(ctx context.Context, req *userapi.TokenRequest) (*userapi.UserReply, error) {
	claims, err := h.service.ValidateToken(ctx, strings.TrimPrefix(req.Token, "Bearer "))
	if err != nil {
		return nil, err
	}
	return &userapi.UserReply{UserId: claims.UserID, Username: claims.Username}, nil
}
