package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"gin-demo/user-service/internal/model"
)

func ReadJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return errors.New("empty request body")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteOK(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, model.Response{
		Code:    http.StatusOK,
		Message: "ok",
		Data:    data,
	})
}

func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, model.Response{
		Code:    http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, model.Response{
		Code:    status,
		Message: message,
	})
}
