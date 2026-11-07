package handler

import (
	"net/http"

	"http-demo/internal/httpapi"
	"http-demo/internal/model"
	"http-demo/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	user, err := h.auth.Register(r.Context(), req)
	if err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusCreated, "created", user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	resp, err := h.auth.Login(r.Context(), req)
	if err != nil {
		httpapi.WriteResponse(w, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", resp)
}
