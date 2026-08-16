// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/logic"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/svc"
	"github.com/zzxrepository/gocode-examples/go/01-go-zero-demo/gateway/internal/types"
)

func DeletePostHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), "Authorization", r.Header.Get("Authorization")))
		var req types.IDRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewDeletePostLogic(r.Context(), svcCtx)
		err := l.DeletePost(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
