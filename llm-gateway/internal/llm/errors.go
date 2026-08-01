package llm

import "fmt"

// APIError 表示调用 LLM Provider 接口时返回的 HTTP/API 错误。
type APIError struct {
	Provider   string
	StatusCode int
	Message    string
}

// Error 返回格式化的错误信息，包含 Provider 名称、状态码和消息。
func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.StatusCode == 0 {
		return fmt.Sprintf("%s: %s", e.Provider, e.Message)
	}
	return fmt.Sprintf("%s: HTTP %d: %s", e.Provider, e.StatusCode, e.Message)
}
