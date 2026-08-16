package middleware

import (
	"net/http"
	"strings"

	"gin-demo/internal/model"
	"gin-demo/internal/service"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "user_id"

func Auth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{Code: http.StatusUnauthorized, Message: "missing bearer token"})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := authService.ParseToken(c.Request.Context(), token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.Response{Code: http.StatusUnauthorized, Message: err.Error()})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) int64 {
	value, exists := c.Get(UserIDKey)
	if !exists {
		return 0
	}
	userID, ok := value.(int64)
	if !ok {
		return 0
	}
	return userID
}
