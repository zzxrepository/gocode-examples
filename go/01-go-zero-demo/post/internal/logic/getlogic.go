package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/svc"
	pb_postpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLogic {
	return &GetLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetLogic) Get(in *pb_postpb.PostIDRequest) (*pb_postpb.PostReply, error) {
	post, err := l.svcCtx.Get(l.ctx, in.GetId())
	if err != nil {
		return nil, fmt.Errorf("get post: %w", err)
	}
	return &pb_postpb.PostReply{Id: post.ID, UserId: post.UserID, Title: post.Title, Content: post.Content}, nil
}
