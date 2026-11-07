package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"http-demo/internal/httpapi"
	"http-demo/internal/middleware"
	"http-demo/internal/model"
	"http-demo/internal/service"
)

type PostHandler struct{ posts *service.PostService }

func NewPostHandler(posts *service.PostService) *PostHandler { return &PostHandler{posts: posts} }

func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreatePostRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	post, err := h.posts.Create(r.Context(), middleware.CurrentUserID(r.Context()), req)
	if err != nil {
		httpapi.WriteResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusCreated, "created", post)
}

func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts.List(r.Context())
	if err != nil {
		httpapi.WriteResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", posts)
}

func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	post, err := h.posts.FindByID(r.Context(), id)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", post)
}

func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req model.UpdatePostRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	post, err := h.posts.Update(r.Context(), middleware.CurrentUserID(r.Context()), id, req)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", post)
}

func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.posts.Delete(r.Context(), middleware.CurrentUserID(r.Context()), id); err != nil {
		writeSQLError(w, err)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "deleted", nil)
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpapi.WriteResponse(w, http.StatusBadRequest, "invalid id", nil)
		return 0, false
	}
	return id, true
}

func writeSQLError(w http.ResponseWriter, err error) {
	if err == sql.ErrNoRows {
		httpapi.WriteResponse(w, http.StatusNotFound, "not found", nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusInternalServerError, err.Error(), nil)
}
