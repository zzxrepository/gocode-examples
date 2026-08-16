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

type ListPostsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPostsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPostsLogic {
	return &ListPostsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListPostsLogic) ListPosts() (resp *types.PostListResponse, err error) {
	reply, err := l.svcCtx.Post.List(l.ctx, &postclient.ListRequest{})
	if err != nil {
		return nil, err
	}
	resp = &types.PostListResponse{Posts: make([]types.PostResponse, 0, len(reply.Posts))}
	for _, post := range reply.Posts {
		resp.Posts = append(resp.Posts, types.PostResponse{ID: post.Id, UserID: post.UserId, Title: post.Title, Content: post.Content})
	}
	return resp, nil
}
