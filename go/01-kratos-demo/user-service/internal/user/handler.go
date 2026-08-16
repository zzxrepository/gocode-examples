package user

import (
	"net/http"
	"strings"

	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/httpx"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/model"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/internal/ratelimit"
)

type Handler struct {
	service     *Service
	authLimiter *ratelimit.Limiter
}

func NewHandler(service *Service, authLimiter *ratelimit.Limiter) *Handler {
	return &Handler{service: service, authLimiter: authLimiter}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /internal/v1/auth/register", h.authLimiter.Handler(h.register))
	mux.HandleFunc("POST /internal/v1/auth/login", h.authLimiter.Handler(h.login))
	mux.HandleFunc("POST /internal/v1/auth/validate", h.validateToken)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteOK(w, map[string]string{"service": "user-service", "status": "ok"})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Username == "" || len(req.Password) < 6 {
		httpx.WriteError(w, http.StatusBadRequest, "username is required and password must be at least 6 chars")
		return
	}

	user, err := h.service.Register(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteCreated(w, user)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	httpx.WriteOK(w, resp)
}

func (h *Handler) validateToken(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	claims, err := h.service.ValidateToken(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	httpx.WriteOK(w, claims)
}
