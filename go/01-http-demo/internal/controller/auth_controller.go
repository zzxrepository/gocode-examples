package controller

import (
	"net/http"

	"gin-demo-v1/internal/httpapi"
	"gin-demo-v1/internal/model"
	"gin-demo-v1/internal/service"
)

type AuthController struct{ auth *service.AuthService }

func NewAuthController(auth *service.AuthService) *AuthController { return &AuthController{auth: auth} }

func (ctl *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	user, err := ctl.auth.Register(r.Context(), req)
	if err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusCreated, "created", user)
}

func (ctl *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	resp, err := ctl.auth.Login(r.Context(), req)
	if err != nil {
		httpapi.WriteResponse(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", resp)
}
