package response

import (
	"encoding/json"
	"net/http"

	"github.com/mmzhang/aiagent-eino-backend/internal/middleware"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	TraceID string `json:"trace_id"`
}

func WriteJSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func OK(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data, TraceID: middleware.TraceID(r.Context())})
}

func Error(w http.ResponseWriter, r *http.Request, status, code int, message string) {
	WriteJSON(w, status, Envelope{Code: code, Message: message, Data: nil, TraceID: middleware.TraceID(r.Context())})
}

func WriteSSE(w http.ResponseWriter, r *http.Request, event string, code int, message string, data any) {
	payload, err := json.Marshal(Envelope{Code: code, Message: message, Data: data, TraceID: middleware.TraceID(r.Context())})
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("event: " + event + "\ndata: " + string(payload) + "\n\n"))
}
