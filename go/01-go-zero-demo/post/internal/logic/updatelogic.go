package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/svc"
	pb_postpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLogic {
	return &UpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateLogic) Update(in *pb_postpb.UpdateRequest) (*pb_postpb.PostReply, error) {
	post, err := l.svcCtx.Update(l.ctx, in.GetUserId(), in.GetId(), in.GetTitle(), in.GetContent())
	if err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}
	return &pb_postpb.PostReply{Id: post.ID, UserId: post.UserID, Title: post.Title, Content: post.Content}, nil
}
