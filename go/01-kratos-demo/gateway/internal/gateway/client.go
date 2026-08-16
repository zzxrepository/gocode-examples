package gateway

import (
	"context"

	postapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/api"
	userapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/api"

	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
)

type UserClient struct{ client userapi.UserClient }
type PostClient struct{ client postapi.PostClient }

func NewUserClient(endpoint string) (*UserClient, error) {
	conn, err := kgrpc.DialInsecure(context.Background(), kgrpc.WithEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return &UserClient{client: userapi.NewUserClient(conn)}, nil
}

func NewPostClient(endpoint string) (*PostClient, error) {
	conn, err := kgrpc.DialInsecure(context.Background(), kgrpc.WithEndpoint(endpoint))
	if err != nil {
		return nil, err
	}
	return &PostClient{client: postapi.NewPostClient(conn)}, nil
}

func (c *UserClient) Register(ctx context.Context, username, password string) (*userapi.UserReply, error) {
	return c.client.Register(ctx, &userapi.RegisterRequest{Username: username, Password: password})
}

func (c *UserClient) Login(ctx context.Context, username, password string) (*userapi.TokenReply, error) {
	return c.client.Login(ctx, &userapi.LoginRequest{Username: username, Password: password})
}

func (c *UserClient) ValidateToken(ctx context.Context, token string) (*userapi.UserReply, error) {
	return c.client.ValidateToken(ctx, &userapi.TokenRequest{Token: token})
}

func (c *PostClient) List(ctx context.Context) (*postapi.PostListReply, error) {
	return c.client.List(ctx, &postapi.ListRequest{})
}

func (c *PostClient) Get(ctx context.Context, id int64) (*postapi.PostReply, error) {
	return c.client.Get(ctx, &postapi.PostIDRequest{Id: id})
}

func (c *PostClient) Create(ctx context.Context, userID int64, title, content string) (*postapi.PostReply, error) {
	return c.client.Create(ctx, &postapi.CreateRequest{UserId: userID, Title: title, Content: content})
}

func (c *PostClient) Update(ctx context.Context, userID, id int64, title, content string) (*postapi.PostReply, error) {
	return c.client.Update(ctx, &postapi.UpdateRequest{UserId: userID, Id: id, Title: title, Content: content})
}

func (c *PostClient) Delete(ctx context.Context, userID, id int64) error {
	_, err := c.client.Delete(ctx, &postapi.DeleteRequest{UserId: userID, Id: id})
	return err
}
