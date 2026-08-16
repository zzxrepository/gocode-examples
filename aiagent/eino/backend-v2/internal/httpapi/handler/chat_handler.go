package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/mmzhang/aiagent-eino-v2-backend/internal/agent"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/domain"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/httpapi/response"
)

type ChatHandler struct {
	agentService *agent.Service
}

func NewChatHandler(agentService *agent.Service) *ChatHandler {
	return &ChatHandler{agentService: agentService}
}

func (h *ChatHandler) Health(w http.ResponseWriter, r *http.Request) {
	defaultModel, _ := h.agentService.Models()
	response.OK(w, r, map[string]string{"status": "ok", "default_model": defaultModel})
}

func (h *ChatHandler) Models(w http.ResponseWriter, r *http.Request) {
	defaultModel, models := h.agentService.Models()
	response.OK(w, r, map[string]any{"default_model": defaultModel, "models": models})
}

func (h *ChatHandler) StreamChat(w http.ResponseWriter, r *http.Request) {
	request, err := decodeRequest(w, r)
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, 40000, "请求格式错误")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		response.Error(w, r, http.StatusInternalServerError, 50001, "当前服务器不支持流式响应")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	err = h.agentService.Stream(r.Context(), request, func(delta string) error {
		response.WriteSSE(w, r, "delta", 0, "ok", map[string]string{"delta": delta})
		flusher.Flush()
		return nil
	})
	if err != nil {
		if serviceError, ok := agent.AsError(err); ok {
			log.Printf("trace_id=%s agent stream error: %v", response.TraceID(r.Context()), serviceError.Cause)
			response.WriteSSE(w, r, "error", serviceError.Code, serviceError.Message, nil)
		} else {
			log.Printf("trace_id=%s unexpected agent error: %v", response.TraceID(r.Context()), err)
			response.WriteSSE(w, r, "error", 50001, "服务器内部错误", nil)
		}
		flusher.Flush()
		return
	}
	response.WriteSSE(w, r, "done", 0, "ok", map[string]bool{"done": true})
	flusher.Flush()
}

func decodeRequest(w http.ResponseWriter, r *http.Request) (domain.ChatRequest, error) {
	var request domain.ChatRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return domain.ChatRequest{}, err
	}
	return request, nil
}
