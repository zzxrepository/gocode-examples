package response

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	TraceID string `json:"trace_id"`
}

type traceIDKey struct{}

func WithTraceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = newTraceID()
		}
		w.Header().Set("X-Trace-ID", traceID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), traceIDKey{}, traceID)))
	})
}

func TraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey{}).(string); ok {
		return traceID
	}
	return ""
}

func WriteJSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func OK(w http.ResponseWriter, r *http.Request, data any) {
	WriteJSON(w, http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data, TraceID: TraceID(r.Context())})
}

func Error(w http.ResponseWriter, r *http.Request, status, code int, message string) {
	WriteJSON(w, status, Envelope{Code: code, Message: message, Data: nil, TraceID: TraceID(r.Context())})
}

func WriteSSE(w http.ResponseWriter, r *http.Request, event string, code int, message string, data any) {
	payload, err := json.Marshal(Envelope{Code: code, Message: message, Data: data, TraceID: TraceID(r.Context())})
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("event: " + event + "\ndata: " + string(payload) + "\n\n"))
}

func newTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "trace-id-unavailable"
	}
	return hex.EncodeToString(bytes)
}
