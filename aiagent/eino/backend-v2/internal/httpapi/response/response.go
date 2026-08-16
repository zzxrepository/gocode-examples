package response

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

// Envelope is used by both ordinary HTTP APIs and every SSE event, so clients
// have one error/data contract even though the response transport differs.
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
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return traceID
}

func OK(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, http.StatusOK, Envelope{Code: 0, Message: "ok", Data: data, TraceID: TraceID(r.Context())})
}

func Error(w http.ResponseWriter, r *http.Request, status, code int, message string) {
	writeJSON(w, status, Envelope{Code: code, Message: message, Data: nil, TraceID: TraceID(r.Context())})
}

func WriteSSE(w http.ResponseWriter, r *http.Request, event string, code int, message string, data any) {
	payload, err := json.Marshal(Envelope{Code: code, Message: message, Data: data, TraceID: TraceID(r.Context())})
	if err == nil {
		_, _ = w.Write([]byte("event: " + event + "\ndata: " + string(payload) + "\n\n"))
	}
}

func writeJSON(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func newTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "trace-id-unavailable"
	}
	return hex.EncodeToString(bytes)
}
