// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/logic"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/svc"
)

func ListPostsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewListPostsLogic(r.Context(), svcCtx)
		resp, err := l.ListPosts()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
