// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package logic

import (
	"context"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/svc"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/types"
	postclient "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/post"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePostLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeletePostLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePostLogic {
	return &DeletePostLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeletePostLogic) DeletePost(req *types.IDRequest) error {
	user, err := currentUser(l.ctx, l.svcCtx)
	if err != nil {
		return err
	}
	_, err = l.svcCtx.Post.Delete(l.ctx, &postclient.DeleteRequest{UserId: user.UserId, Id: req.ID})
	return err
}
