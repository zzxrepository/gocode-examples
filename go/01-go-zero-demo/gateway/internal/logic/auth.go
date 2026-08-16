package logic

import (
	"context"
	"fmt"
	"strings"

	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/svc"
	userclient "github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/user/user"
)

func currentUser(ctx context.Context, service *svc.ServiceContext) (*userclient.UserReply, error) {
	token, ok := ctx.Value("Authorization").(string)
	if !ok || !strings.HasPrefix(token, "Bearer ") {
		return nil, fmt.Errorf("authorization header is required")
	}
	return service.User.ValidateToken(ctx, &userclient.TokenRequest{Token: strings.TrimPrefix(token, "Bearer ")})
}
