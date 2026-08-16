package logic

import (
	"context"
	"fmt"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/internal/svc"
	pb_userpb "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/pb/github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ValidateTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewValidateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ValidateTokenLogic {
	return &ValidateTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ValidateTokenLogic) ValidateToken(in *pb_userpb.TokenRequest) (*pb_userpb.UserReply, error) {
	user, err := l.svcCtx.ValidateToken(l.ctx, in.GetToken())
	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}
	return &pb_userpb.UserReply{UserId: user.ID, Email: user.Email}, nil
}
