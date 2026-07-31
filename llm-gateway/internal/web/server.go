package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-in-action/llm-gateway/internal/cost"
	"agent-in-action/llm-gateway/internal/llm"
	"agent-in-action/llm-gateway/internal/router"
)

// Server 暴露 LLM 网关的 HTTP 接口。
type Server struct {
	router      *router.Router
	timeout     time.Duration
	corsOrigins []string
}

// Option 配置 Server。
type Option func(*Server)

// WithTimeout 设置每个请求的最大处理时间。
func WithTimeout(timeout time.Duration) Option {
	return func(s *Server) { s.timeout = timeout }
}

// WithCORS 设置允许的 Origin 列表；传入 ["*"] 表示允许任意 Origin。
func WithCORS(origins []string) Option {
	return func(s *Server) { s.corsOrigins = origins }
}

// New 创建 HTTP Server。
func New(router *router.Router, opts ...Option) *Server {
	server := &Server{
		router:      router,
		timeout:     60 * time.Second,
		corsOrigins: nil,
	}
	for _, opt := range opts {
		opt(server)
	}
	return server
}

// Handler 返回注册好路由的 http.Handler。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat", s.handleChat)
	mux.HandleFunc("/health", s.handleHealth)

	var handler http.Handler = mux
	handler = s.timeoutMiddleware(handler)
	handler = s.corsMiddleware(handler)
	return handler
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// chatRequest 复用 llm.ChatRequest，不需要额外包装。
type chatRequest = llm.ChatRequest

type chatResponse struct {
	Provider     string   `json:"provider"`
	Model        string   `json:"model,omitempty"`
	Content      string   `json:"content"`
	FinishReason string   `json:"finish_reason,omitempty"`
	Usage        usage    `json:"usage"`
	Cost         costInfo `json:"cost"`
	DurationMs   int64    `json:"duration_ms"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type costInfo struct {
	Estimated float64 `json:"estimated"`
	Currency  string  `json:"currency"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if err := llm.ValidateRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()

	if req.Stream {
		s.handleStream(ctx, w, req)
		return
	}

	s.handleChatSync(ctx, w, req)
}

func (s *Server) handleChatSync(ctx context.Context, w http.ResponseWriter, req llm.ChatRequest) {
	result, err := s.router.Chat(ctx, req)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			status = http.StatusGatewayTimeout
		}
		writeError(w, status, fmt.Sprintf("chat failed: %v", err))
		return
	}

	response := chatResponse{
		Provider:     result.Provider,
		Model:        result.Response.Model,
		Content:      result.Response.Content,
		FinishReason: result.Response.FinishReason,
		Usage: usage{
			InputTokens:  result.Response.Usage.InputTokens,
			OutputTokens: result.Response.Usage.OutputTokens,
			TotalTokens:  result.Response.Usage.TotalTokens(),
		},
		Cost: costInfo{
			Estimated: cost.Estimate(result.Response.Usage, result.Pricing),
			Currency:  result.Pricing.Currency,
		},
		DurationMs: result.Duration.Milliseconds(),
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleStream(ctx context.Context, w http.ResponseWriter, req llm.ChatRequest) {
	startedAt := time.Now()
	result, err := s.router.ChatStream(ctx, req)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			status = http.StatusGatewayTimeout
		}
		writeError(w, status, fmt.Sprintf("chat stream failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flush, ok := w.(http.Flusher)
	if !ok {
		_ = writeSSE(w, []byte(`{"done":true,"error":"streaming not supported by response writer"}`))
		return
	}

	var usageAcc llm.Usage
	for chunk := range result.Chunks {
		if chunk.Err != nil {
			_ = writeSSE(w, []byte(fmt.Sprintf(`{"done":true,"error":%q}`, chunk.Err.Error())))
			flush.Flush()
			return
		}

		if chunk.Content != "" {
			event := map[string]any{
				"provider": result.Provider,
				"content":  chunk.Content,
			}
			if err := writeSSEEvent(w, event); err != nil {
				return
			}
			flush.Flush()
		}

		if chunk.Usage != nil {
			usageAcc = *chunk.Usage
		}
	}

	finalEvent := map[string]any{
		"provider": result.Provider,
		"done":     true,
		"usage": usage{
			InputTokens:  usageAcc.InputTokens,
			OutputTokens: usageAcc.OutputTokens,
			TotalTokens:  usageAcc.TotalTokens(),
		},
		"cost": costInfo{
			Estimated: cost.Estimate(usageAcc, result.Pricing),
			Currency:  result.Pricing.Currency,
		},
		"duration_ms": time.Since(startedAt).Milliseconds(),
	}

	_ = writeSSEEvent(w, finalEvent)
	flush.Flush()
}

func writeSSEEvent(w io.Writer, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return writeSSE(w, data)
}

func writeSSE(w io.Writer, data []byte) error {
	_, err := fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// timeoutMiddleware 为每个请求注入超时 context。
func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// corsMiddleware 根据配置的 Origin 列表添加 CORS 头。
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	if len(s.corsOrigins) == 0 {
		return next
	}

	allowAll := false
	for _, origin := range s.corsOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && contains(s.corsOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}
