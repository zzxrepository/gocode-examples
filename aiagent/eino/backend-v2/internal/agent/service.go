// Package agent is the application service layer. It adapts our HTTP-facing
// request into Eino's ChatModelAgent + Runner execution model.
package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/domain"
	"github.com/mmzhang/aiagent-eino-v2-backend/internal/modelregistry"
)

type Service struct {
	registry     *modelregistry.Registry
	defaultModel string
}

func NewService(registry *modelregistry.Registry, defaultModel string) *Service {
	return &Service{registry: registry, defaultModel: defaultModel}
}

func (s *Service) Models() (string, []domain.ModelInfo) {
	models := s.registry.Models()
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	return s.defaultModel, models
}

func (s *Service) Stream(ctx context.Context, request domain.ChatRequest, onDelta func(string) error) error {
	messages, err := toEinoMessages(request.Messages)
	if err != nil {
		return err
	}
	modelRef := request.Model
	if modelRef == "" {
		modelRef = s.defaultModel
	}
	target, ok := s.registry.Resolve(modelRef)
	if !ok {
		return NewError(400, 40003, "所选模型不在允许的模型列表中", nil)
	}
	temperature := target.DefaultTemperature
	if request.Temperature != nil {
		temperature = *request.Temperature
	}
	if temperature < 0 || temperature > 2 {
		return NewError(400, 40004, "temperature 必须在 0 到 2 之间", nil)
	}

	chatModel, err := target.NewChatModel(ctx, temperature)
	if err != nil {
		return NewError(502, 50201, "初始化模型失败", err)
	}
	chatAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "chat",
		Description:   "流式多轮对话助手",
		Instruction:   target.SystemPrompt,
		Model:         chatModel,
		MaxIterations: 1, // v2 先只做对话；增加 Tools 后可调高以启用 ReAct 循环。
	})
	if err != nil {
		return NewError(500, 50001, "构建 Eino ChatModelAgent 失败", err)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: chatAgent, EnableStreaming: true})
	iterator := runner.Run(ctx, messages)
	for {
		event, ok := iterator.Next()
		if !ok {
			return nil
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return NewError(502, 50202, "模型流中断", event.Err)
		}
		if err := forwardMessageEvent(event, onDelta); err != nil {
			return NewError(502, 50202, "模型流中断", err)
		}
	}
}

func toEinoMessages(messages []domain.Message) ([]*schema.Message, error) {
	if len(messages) == 0 {
		return nil, NewError(400, 40001, "messages 不能为空", nil)
	}
	result := make([]*schema.Message, 0, len(messages))
	for _, message := range messages {
		if strings.TrimSpace(message.Content) == "" {
			return nil, NewError(400, 40002, "消息内容不能为空", nil)
		}
		switch message.Role {
		case "user":
			result = append(result, schema.UserMessage(message.Content))
		case "assistant":
			result = append(result, schema.AssistantMessage(message.Content, nil))
		default:
			return nil, NewError(400, 40002, "消息仅支持 user 或 assistant 角色", nil)
		}
	}
	return result, nil
}

func forwardMessageEvent(event *adk.AgentEvent, onDelta func(string) error) error {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil
	}
	output := event.Output.MessageOutput
	if output.Role != schema.Assistant {
		return nil
	}
	if !output.IsStreaming {
		if output.Message != nil && output.Message.Content != "" {
			return onDelta(output.Message.Content)
		}
		return nil
	}
	if output.MessageStream == nil {
		return errors.New("Eino Agent 事件缺少 MessageStream")
	}
	defer output.MessageStream.Close()
	for {
		chunk, err := output.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取 Eino MessageStream 失败：%w", err)
		}
		if chunk != nil && chunk.Content != "" {
			if err := onDelta(chunk.Content); err != nil {
				return err
			}
		}
	}
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
	return serviceError, errors.As(err, &serviceError)
}
