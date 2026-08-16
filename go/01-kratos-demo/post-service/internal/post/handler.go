package post

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/httpx"
	"github.com/zzxrepository/gocode-examples/go/01-kratos-demo/post-service/internal/model"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /internal/v1/posts", h.list)
	mux.HandleFunc("GET /internal/v1/posts/{id}", h.get)
	mux.HandleFunc("POST /internal/v1/posts", h.create)
	mux.HandleFunc("PUT /internal/v1/posts/{id}", h.update)
	mux.HandleFunc("DELETE /internal/v1/posts/{id}", h.delete)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteOK(w, map[string]string{"service": "post-service", "status": "ok"})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromHeader(w, r)
	if !ok {
		return
	}
	var req model.CreatePostRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Title == "" || req.Content == "" {
		httpx.WriteError(w, http.StatusBadRequest, "title and content are required")
		return
	}

	post, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteCreated(w, post)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	posts, err := h.service.List(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteOK(w, posts)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}
	post, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpx.WriteOK(w, post)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromHeader(w, r)
	if !ok {
		return
	}
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}
	var req model.UpdatePostRequest
	if err := httpx.ReadJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	post, err := h.service.Update(r.Context(), userID, id, req)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpx.WriteOK(w, post)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromHeader(w, r)
	if !ok {
		return
	}
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), userID, id); err != nil {
		writeSQLError(w, err)
		return
	}
	httpx.WriteOK(w, nil)
}

func userIDFromHeader(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	if err != nil || userID <= 0 {
		httpx.WriteError(w, http.StatusUnauthorized, "missing user id")
		return 0, false
	}
	return userID, true
}

func idFromPath(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeSQLError(w http.ResponseWriter, err error) {
	if err == sql.ErrNoRows {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, err.Error())
}
