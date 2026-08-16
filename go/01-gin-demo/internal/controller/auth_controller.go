package controller

import (
	"net/http"

	"gin-demo/internal/model"
	"gin-demo/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthController is the HTTP adapter for authentication use cases.
// It binds HTTP input, invokes AuthService, and writes HTTP responses.
type AuthController struct {
	auth *service.AuthService
}

func NewAuthController(auth *service.AuthService) *AuthController {
	return &AuthController{auth: auth}
}

func (ctl *AuthController) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	user, err := ctl.auth.Register(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, model.Response{Code: http.StatusCreated, Message: "created", Data: user})
}

func (ctl *AuthController) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	resp, err := ctl.auth.Login(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.Response{Code: http.StatusUnauthorized, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: http.StatusOK, Message: "ok", Data: resp})
}
