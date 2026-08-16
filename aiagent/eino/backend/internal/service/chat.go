package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mmzhang/aiagent-eino-backend/internal/domain"
	"github.com/mmzhang/aiagent-eino-backend/internal/provider"
)

type ChatService struct {
	registry     *provider.Registry
	defaultModel string
}

func NewChatService(registry *provider.Registry, defaultModel string) *ChatService {
	return &ChatService{registry: registry, defaultModel: defaultModel}
}

func (s *ChatService) Models() (string, []domain.ModelInfo) {
	models := s.registry.Models()
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return s.defaultModel, models
}

func (s *ChatService) Stream(ctx context.Context, request domain.ChatRequest, onDelta func(string) error) error {
	if len(request.Messages) == 0 {
		return NewError(400, 40001, "messages 不能为空", nil)
	}
	for _, message := range request.Messages {
		if (message.Role != "user" && message.Role != "assistant") || strings.TrimSpace(message.Content) == "" {
			return NewError(400, 40002, "消息仅支持非空的 user 或 assistant 内容", nil)
		}
	}

	modelRef := request.Model
	if modelRef == "" {
		modelRef = s.defaultModel
	}
	target, ok := s.registry.Resolve(modelRef)
	if !ok {
		return NewError(400, 40003, "所选模型不在允许的模型列表中", nil)
	}
	temperature := target.Provider.DefaultTemperature()
	if request.Temperature != nil {
		temperature = *request.Temperature
	}
	if temperature < 0 || temperature > 2 {
		return NewError(400, 40004, "temperature 必须在 0 到 2 之间", nil)
	}
	if err := target.Provider.Stream(ctx, target.Info.Model, request.Messages, temperature, onDelta); err != nil {
		if remote, ok := provider.IsRemoteError(err); ok {
			return NewError(502, 50201, "模型服务请求失败", remote)
		}
		return NewError(502, 50202, "模型流中断", err)
	}
	return nil
}

type Error struct {
	Status  int
	Code    int
	Message string
	Cause   error
}

func NewError(status, code int, message string, cause error) *Error {
	return &Error{Status: status, Code: code, Message: message, Cause: cause}
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }

func AsError(err error) (*Error, bool) {
	var serviceError *Error
	if errors.As(err, &serviceError) {
		return serviceError, true
	}
	return nil, false
}

func DescribeError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%+v", err)
}
