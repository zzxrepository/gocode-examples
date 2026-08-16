package logic

import (
	"context"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/svc"
	pb_postpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListLogic) List(in *pb_postpb.ListRequest) (*pb_postpb.PostListReply, error) {
	posts, err := l.svcCtx.List(l.ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*pb_postpb.PostReply, 0, len(posts))
	for _, post := range posts {
		result = append(result, &pb_postpb.PostReply{Id: post.ID, UserId: post.UserID, Title: post.Title, Content: post.Content})
	}
	return &pb_postpb.PostListReply{Posts: result}, nil
}
