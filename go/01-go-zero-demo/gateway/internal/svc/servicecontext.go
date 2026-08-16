// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package svc

import (
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/config"
	postclient "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/post/post"
	userclient "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/user"
)

type ServiceContext struct {
	Config config.Config
	User   userclient.User
	Post   postclient.Post
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		User:   userclient.NewUser(zrpc.MustNewClient(c.UserRpc)),
		Post:   postclient.NewPost(zrpc.MustNewClient(c.PostRpc)),
	}
}
