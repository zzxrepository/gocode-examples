package controller

import (
	"database/sql"
	"net/http"
	"strconv"

	"gin-demo/internal/middleware"
	"gin-demo/internal/model"
	"gin-demo/internal/service"

	"github.com/gin-gonic/gin"
)

// PostController is the HTTP adapter for post-related use cases.
type PostController struct {
	posts *service.PostService
}

func NewPostController(posts *service.PostService) *PostController {
	return &PostController{posts: posts}
}

func (ctl *PostController) Create(c *gin.Context) {
	var req model.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	post, err := ctl.posts.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, model.Response{Code: http.StatusCreated, Message: "created", Data: post})
}

func (ctl *PostController) List(c *gin.Context) {
	posts, err := ctl.posts.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: http.StatusOK, Message: "ok", Data: posts})
}

func (ctl *PostController) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	post, err := ctl.posts.FindByID(c.Request.Context(), id)
	if err != nil {
		writeSQLError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: http.StatusOK, Message: "ok", Data: post})
}

func (ctl *PostController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req model.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Code: http.StatusBadRequest, Message: err.Error()})
		return
	}
	post, err := ctl.posts.Update(c.Request.Context(), middleware.CurrentUserID(c), id, req)
	if err != nil {
		writeSQLError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: http.StatusOK, Message: "ok", Data: post})
}

func (ctl *PostController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := ctl.posts.Delete(c.Request.Context(), middleware.CurrentUserID(c), id); err != nil {
		writeSQLError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.Response{Code: http.StatusOK, Message: "deleted"})
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, model.Response{Code: http.StatusBadRequest, Message: "invalid id"})
		return 0, false
	}
	return id, true
}

func writeSQLError(c *gin.Context, err error) {
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, model.Response{Code: http.StatusNotFound, Message: "not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, model.Response{Code: http.StatusInternalServerError, Message: err.Error()})
}
