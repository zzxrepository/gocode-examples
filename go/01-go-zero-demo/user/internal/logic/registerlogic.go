package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/internal/svc"
	pb_userpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *pb_userpb.RegisterRequest) (*pb_userpb.UserReply, error) {
	if in.GetEmail() == "" || in.GetPassword() == "" {
		return nil, fmt.Errorf("email and password are required")
	}
	user, err := l.svcCtx.Register(l.ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		return nil, fmt.Errorf("register user: %w", err)
	}
	return &pb_userpb.UserReply{UserId: user.ID, Email: user.Email}, nil
}
