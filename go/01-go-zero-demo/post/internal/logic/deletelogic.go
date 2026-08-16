package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/internal/svc"
	pb_postpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteLogic {
	return &DeleteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteLogic) Delete(in *pb_postpb.DeleteRequest) (*pb_postpb.Empty, error) {
	if err := l.svcCtx.Delete(l.ctx, in.GetUserId(), in.GetId()); err != nil {
		return nil, fmt.Errorf("delete post: %w", err)
	}
	return &pb_postpb.Empty{}, nil
}
