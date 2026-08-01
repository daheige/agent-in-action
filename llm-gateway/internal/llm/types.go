package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Role 表示消息发送者的角色。
type Role string

const (
	// RoleSystem 系统角色，用于设定全局行为。
	RoleSystem Role = "system"
	// RoleUser 用户角色。
	RoleUser Role = "user"
	// RoleAssistant 助手角色。
	RoleAssistant Role = "assistant"
	// RoleTool 工具角色，用于工具调用结果。
	RoleTool Role = "tool"
)

// Message 表示一条聊天消息。
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 表示一次聊天请求。
type ChatRequest struct {
	// Model 为空时由每个 Provider 使用自己的默认模型，便于跨 Provider 故障转移。
	Model       string    `json:"model,omitempty"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// Option 用于构造 ChatRequest 的可选配置函数。
type Option func(*ChatRequest)

// NewChatRequest 使用指定的模型和消息创建 ChatRequest，并应用可选配置。
func NewChatRequest(model string, messages []Message, opts ...Option) ChatRequest {
	req := ChatRequest{
		Model:    model,
		Messages: append([]Message(nil), messages...),
	}
	for _, opt := range opts {
		opt(&req)
	}
	return req
}

// WithTemperature 设置聊天请求的采样温度。
func WithTemperature(temperature float64) Option {
	return func(req *ChatRequest) { req.Temperature = &temperature }
}

// WithMaxTokens 设置聊天请求的最大输出 token 数。
func WithMaxTokens(maxTokens int) Option {
	return func(req *ChatRequest) { req.MaxTokens = &maxTokens }
}

// Usage 表示一次调用的 token 使用量。
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// TotalTokens 返回输入与输出 token 的总和。
func (u Usage) TotalTokens() int {
	return u.InputTokens + u.OutputTokens
}

// ChatResponse 表示一次非流式聊天的完整响应。
type ChatResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        Usage
}

// StreamChunk 表示流式输出中的一个数据块。
type StreamChunk struct {
	Content string
	Usage   *Usage
	Done    bool
	Err     error
}

// Capability 描述 Provider 支持的能力。
type Capability struct {
	Streaming bool
	Thinking  bool
	Tools     bool
}

// Provider 定义 LLM Provider 需要实现的接口。
type Provider interface {
	Name() string
	DefaultModel() string
	Capabilities() Capability
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// ValidateRequest 校验 ChatRequest 是否合法。
func ValidateRequest(req ChatRequest) error {
	if len(req.Messages) == 0 {
		return errors.New("messages 不能为空")
	}
	for i, message := range req.Messages {
		switch message.Role {
		case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		default:
			return fmt.Errorf("messages[%d] 使用了未知 role %q", i, message.Role)
		}
		if strings.TrimSpace(message.Content) == "" {
			return fmt.Errorf("messages[%d].content 不能为空", i)
		}
	}
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return fmt.Errorf("temperature 必须在 [0, 2] 范围内")
	}
	if req.MaxTokens != nil && *req.MaxTokens <= 0 {
		return errors.New("max_tokens 必须大于 0")
	}
	return nil
}

// EffectiveModel 返回请求中指定的模型或 Provider 默认模型，两者都为空时返回错误。
func EffectiveModel(requestModel, defaultModel string) (string, error) {
	if model := strings.TrimSpace(requestModel); model != "" {
		return model, nil
	}
	if model := strings.TrimSpace(defaultModel); model != "" {
		return model, nil
	}
	return "", errors.New("请求和 Provider 均未配置模型")
}
