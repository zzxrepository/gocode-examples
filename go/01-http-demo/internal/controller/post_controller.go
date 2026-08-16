package controller

import (
	"database/sql"
	"net/http"
	"strconv"

	"gin-demo-v1/internal/httpapi"
	"gin-demo-v1/internal/middleware"
	"gin-demo-v1/internal/model"
	"gin-demo-v1/internal/service"
)

type PostController struct{ posts *service.PostService }

func NewPostController(posts *service.PostService) *PostController { return &PostController{posts: posts} }

func (ctl *PostController) Create(w http.ResponseWriter, r *http.Request) {
	var req model.CreatePostRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	post, err := ctl.posts.Create(r.Context(), middleware.CurrentUserID(r.Context()), req)
	if err != nil {
		httpapi.WriteResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusCreated, "created", post)
}

func (ctl *PostController) List(w http.ResponseWriter, r *http.Request) {
	posts, err := ctl.posts.List(r.Context())
	if err != nil {
		httpapi.WriteResponse(w, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", posts)
}

func (ctl *PostController) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	post, err := ctl.posts.FindByID(r.Context(), id)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", post)
}

func (ctl *PostController) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req model.UpdatePostRequest
	if err := httpapi.DecodeJSON(w, r, &req); err != nil {
		httpapi.WriteResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}
	post, err := ctl.posts.Update(r.Context(), middleware.CurrentUserID(r.Context()), id, req)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	httpapi.WriteResponse(w, http.StatusOK, "ok", post)
}

func (ctl *PostController) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := ctl.posts.Delete(r.Context(), middleware.CurrentUserID(r.Context()), id); err != nil {
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
