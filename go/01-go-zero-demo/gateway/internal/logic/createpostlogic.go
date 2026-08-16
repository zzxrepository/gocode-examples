// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/svc"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/types"
	postclient "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/post"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePostLogic {
	return &CreatePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePostLogic) CreatePost(req *types.PostRequest) (resp *types.PostResponse, err error) {
	user, err := currentUser(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	post, err := l.svcCtx.Post.Create(l.ctx, &postclient.CreateRequest{UserId: user.UserId, Title: req.Title, Content: req.Content})
	if err != nil {
		return nil, err
	}
	return &types.PostResponse{ID: post.Id, UserID: post.UserId, Title: post.Title, Content: post.Content}, nil
}
