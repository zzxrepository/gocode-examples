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

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.TokenResponse, err error) {
	token, err := l.svcCtx.User.Login(l.ctx, &userclient.LoginRequest{Email: req.Email, Password: req.Password})
	if err != nil {
		return nil, err
	}
	return &types.TokenResponse{Token: token.Token, UserID: token.UserId}, nil
}
