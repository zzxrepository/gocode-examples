package gateway

import (
	"io"
	"net/http"

	"gin-demo/gateway/internal/httpx"
)

type Handler struct {
	users *UserClient
	posts *PostClient
}

func NewHandler(users *UserClient, posts *PostClient) *Handler {
	return &Handler{users: users, posts: posts}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("GET /api/v1/posts", h.listPosts)
	mux.HandleFunc("GET /api/v1/posts/{id}", h.getPost)
	mux.HandleFunc("POST /api/v1/posts", h.createPost)
	mux.HandleFunc("PUT /api/v1/posts/{id}", h.updatePost)
	mux.HandleFunc("DELETE /api/v1/posts/{id}", h.deletePost)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteOK(w, map[string]string{"service": "gateway", "status": "ok"})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	resp, err := h.users.Register(r.Context(), r.Body)
	writeDownstream(w, resp, err)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	resp, err := h.users.Login(r.Context(), r.Body)
	writeDownstream(w, resp, err)
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	resp, err := h.posts.List(r.Context())
	writeDownstream(w, resp, err)
}

func (h *Handler) getPost(w http.ResponseWriter, r *http.Request) {
	resp, err := h.posts.Get(r.Context(), r.PathValue("id"))
	writeDownstream(w, resp, err)
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.validate(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.posts.Create(r.Context(), claims.UserID, body)
	writeDownstream(w, resp, err)
}

func (h *Handler) updatePost(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.validate(w, r)
	if !ok {
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.posts.Update(r.Context(), claims.UserID, r.PathValue("id"), body)
	writeDownstream(w, resp, err)
}

func (h *Handler) deletePost(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.validate(w, r)
	if !ok {
		return
	}
	resp, err := h.posts.Delete(r.Context(), claims.UserID, r.PathValue("id"))
	writeDownstream(w, resp, err)
}

func (h *Handler) validate(w http.ResponseWriter, r *http.Request) (*TokenClaimsView, bool) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing authorization header")
		return nil, false
	}

	claims, err := h.users.ValidateToken(r.Context(), authorization)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		return nil, false
	}
	return &TokenClaimsView{UserID: claims.UserID, Username: claims.Username}, true
}

type TokenClaimsView struct {
	UserID   int64
	Username string
}

func writeDownstream(w http.ResponseWriter, resp *http.Response, err error) {
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
