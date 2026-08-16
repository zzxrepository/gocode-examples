// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/svc"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/types"
	userclient "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.TokenResponse, err error) {
	user, err := l.svcCtx.User.Register(l.ctx, &userclient.RegisterRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		return nil, err
	}
	token, err := l.svcCtx.User.Login(l.ctx, &userclient.LoginRequest{Email: user.Email, Password: req.Password})
	if err != nil {
		return nil, err
	}
	return &types.TokenResponse{Token: token.Token, UserID: token.UserId}, nil
}
