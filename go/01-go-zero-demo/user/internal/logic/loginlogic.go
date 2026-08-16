package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/internal/svc"
	pb_userpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *pb_userpb.LoginRequest) (*pb_userpb.TokenReply, error) {
	token, userID, err := l.svcCtx.Login(l.ctx, in.GetEmail(), in.GetPassword())
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	return &pb_userpb.TokenReply{Token: token, UserId: userID}, nil
}
