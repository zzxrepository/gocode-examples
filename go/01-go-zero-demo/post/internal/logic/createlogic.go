package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/svc"
	pb_postpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateLogic {
	return &CreateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateLogic) Create(in *pb_postpb.CreateRequest) (*pb_postpb.PostReply, error) {
	if in.GetUserId() == 0 || in.GetTitle() == "" {
		return nil, fmt.Errorf("user id and title are required")
	}
	post, err := l.svcCtx.Create(l.ctx, in.GetUserId(), in.GetTitle(), in.GetContent())
	if err != nil {
		return nil, err
	}
	return &pb_postpb.PostReply{Id: post.ID, UserId: post.UserID, Title: post.Title, Content: post.Content}, nil
}
