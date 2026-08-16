package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"gin-demo-v1/internal/httpapi"
	"gin-demo-v1/internal/service"
)

type Middleware func(http.Handler) http.Handler

type contextKey string

const userIDKey contextKey = "user_id"

func Chain(next http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("method=%s path=%s duration=%s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered: %v", recovered)
				httpapi.WriteResponse(w, http.StatusInternalServerError, "internal server error", nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func Auth(authService *service.AuthService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				httpapi.WriteResponse(w, http.StatusUnauthorized, "missing bearer token", nil)
				return
			}
			claims, err := authService.ParseToken(r.Context(), strings.TrimPrefix(authHeader, "Bearer "))
			if err != nil {
				httpapi.WriteResponse(w, http.StatusUnauthorized, err.Error(), nil)
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CurrentUserID(ctx context.Context) int64 {
	userID, _ := ctx.Value(userIDKey).(int64)
	return userID
}
