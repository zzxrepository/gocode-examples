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

type GetPostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPostLogic {
	return &GetPostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPostLogic) GetPost(req *types.IDRequest) (resp *types.PostResponse, err error) {
	post, err := l.svcCtx.Post.Get(l.ctx, &postclient.PostIDRequest{Id: req.ID})
	if err != nil {
		return nil, err
	}
	return &types.PostResponse{ID: post.Id, UserID: post.UserId, Title: post.Title, Content: post.Content}, nil
}
