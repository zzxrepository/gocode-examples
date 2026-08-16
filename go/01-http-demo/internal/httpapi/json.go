// Package httpapi contains small, framework-free HTTP helpers.
package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gin-demo-v1/internal/model"
)

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteResponse(w http.ResponseWriter, status int, message string, data any) {
	WriteJSON(w, status, model.Response{Code: status, Message: message, Data: data})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("request body must contain exactly one JSON value")
		}
		return err
	}
	return nil
}
