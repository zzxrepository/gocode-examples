package gateway

import (
	"net/http"
	"strconv"

	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/gateway/internal/httpx"
	userapi "github.com/zzxrepository/gocode-examples/go/01-kratos-demo/user-service/api"
)

type Handler struct {
	users *UserClient
	posts *PostClient
}
type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type postInput struct {
	Title   string `json:"title"`
	Content string `json:"content"`
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

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteOK(w, map[string]string{"service": "gateway", "status": "ok"})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if err := httpx.ReadJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	reply, err := h.users.Register(r.Context(), input.Username, input.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteCreated(w, reply)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var input credentials
	if err := httpx.ReadJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	reply, err := h.users.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		return
	}
	httpx.WriteOK(w, reply)
}

func (h *Handler) listPosts(w http.ResponseWriter, r *http.Request) {
	reply, err := h.posts.List(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.WriteOK(w, reply)
}

func (h *Handler) getPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid post id")
		return
	}
	reply, err := h.posts.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	httpx.WriteOK(w, reply)
}

func (h *Handler) createPost(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	var input postInput
	if err := httpx.ReadJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	reply, err := h.posts.Create(r.Context(), user.UserId, input.Title, input.Content)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteCreated(w, reply)
}

func (h *Handler) updatePost(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid post id")
		return
	}
	var input postInput
	if err := httpx.ReadJSON(r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	reply, err := h.posts.Update(r.Context(), user.UserId, id, input.Title, input.Content)
	if err != nil {
		httpx.WriteError(w, http.StatusForbidden, err.Error())
		return
	}
	httpx.WriteOK(w, reply)
}

func (h *Handler) deletePost(w http.ResponseWriter, r *http.Request) {
	user, ok := h.currentUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid post id")
		return
	}
	if err := h.posts.Delete(r.Context(), user.UserId, id); err != nil {
		httpx.WriteError(w, http.StatusForbidden, err.Error())
		return
	}
	httpx.WriteOK(w, map[string]bool{"deleted": true})
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) (*userapi.UserReply, bool) {
	token := r.Header.Get("Authorization")
	if token == "" {
		httpx.WriteError(w, http.StatusUnauthorized, "missing authorization header")
		return nil, false
	}
	reply, err := h.users.ValidateToken(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, err.Error())
		return nil, false
	}
	return reply, true
}
