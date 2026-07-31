package web

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-in-action/llm-gateway/internal/cost"
	"agent-in-action/llm-gateway/internal/llm"
	"agent-in-action/llm-gateway/internal/router"
)

type fakeProvider struct {
	name         string
	defaultModel string
	response     *llm.ChatResponse
	chunks       []llm.StreamChunk
	err          error
	delay        time.Duration
}

func (p *fakeProvider) Name() string { return p.name }

func (p *fakeProvider) DefaultModel() string { return p.defaultModel }

func (p *fakeProvider) Capabilities() llm.Capability {
	return llm.Capability{Streaming: true}
}

func (p *fakeProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if p.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.delay):
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	if p.response != nil {
		return p.response, nil
	}
	model := p.defaultModel
	if req.Model != "" {
		model = req.Model
	}
	return &llm.ChatResponse{
		Content: "hello from " + p.name,
		Model:   model,
		Usage:   llm.Usage{InputTokens: 2, OutputTokens: 3},
	}, nil
}

func (p *fakeProvider) ChatStream(ctx context.Context, req llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	if p.delay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.delay):
		}
	}
	if p.err != nil {
		return nil, p.err
	}

	output := make(chan llm.StreamChunk, len(p.chunks)+1)
	for _, chunk := range p.chunks {
		output <- chunk
	}
	close(output)
	return output, nil
}

func newTestServer(t *testing.T, candidates ...router.Candidate) *httptest.Server {
	t.Helper()
	modelRouter, err := router.New(router.Priority{}, candidates...)
	if err != nil {
		t.Fatal(err)
	}
	server := New(modelRouter, WithTimeout(5*time.Second), WithCORS([]string{"*"}))
	return httptest.NewServer(server.Handler())
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestChatNonStreaming(t *testing.T) {
	provider := &fakeProvider{
		name:         "fake",
		defaultModel: "fake-model",
		response: &llm.ChatResponse{
			Content:      "hello",
			Model:        "fake-model",
			FinishReason: "stop",
			Usage:        llm.Usage{InputTokens: 2, OutputTokens: 3},
		},
	}
	ts := newTestServer(t, router.Candidate{
		Provider: provider,
		Pricing:  cost.Pricing{InputPer1M: 1, OutputPer1M: 2, Currency: "USD"},
	})
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/chat", map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
		"stream": false,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}

	if result.Provider != "fake" || result.Model != "fake-model" || result.Content != "hello" {
		t.Fatalf("unexpected response: %+v", result)
	}
	if result.Usage.InputTokens != 2 || result.Usage.OutputTokens != 3 || result.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage: %+v", result.Usage)
	}
	if result.Cost.Currency != "USD" || result.Cost.Estimated <= 0 {
		t.Fatalf("unexpected cost: %+v", result.Cost)
	}
}

func TestChatStreaming(t *testing.T) {
	provider := &fakeProvider{
		name:         "fake",
		defaultModel: "fake-model",
		chunks: []llm.StreamChunk{
			{Content: "hello"},
			{Content: " world"},
			{Usage: &llm.Usage{InputTokens: 2, OutputTokens: 3}},
		},
	}
	ts := newTestServer(t, router.Candidate{
		Provider: provider,
		Pricing:  cost.Pricing{InputPer1M: 1, OutputPer1M: 2, Currency: "USD"},
	})
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/chat", map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
		"stream": true,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q", ct)
	}

	scanner := bufio.NewScanner(resp.Body)
	var contents []string
	var final map[string]any
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			t.Fatalf("unexpected line: %q", line)
		}
		data := strings.TrimPrefix(line, "data: ")
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Fatal(err)
		}
		if content, ok := event["content"].(string); ok {
			contents = append(contents, content)
		}
		if done, _ := event["done"].(bool); done {
			final = event
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	if strings.Join(contents, "") != "hello world" {
		t.Fatalf("contents = %v", contents)
	}
	if final == nil {
		t.Fatal("missing final event")
	}
	if final["provider"] != "fake" {
		t.Fatalf("unexpected provider: %v", final["provider"])
	}
}

func TestChatValidationError(t *testing.T) {
	provider := &fakeProvider{name: "fake", defaultModel: "fake-model"}
	ts := newTestServer(t, router.Candidate{Provider: provider})
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/chat", map[string]any{
		"messages": []map[string]string{},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestChatMethodNotAllowed(t *testing.T) {
	provider := &fakeProvider{name: "fake", defaultModel: "fake-model"}
	ts := newTestServer(t, router.Candidate{Provider: provider})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/chat")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestChatAllProvidersFailed(t *testing.T) {
	provider := &fakeProvider{
		name:         "fake",
		defaultModel: "fake-model",
		err:          errors.New("provider down"),
	}
	ts := newTestServer(t, router.Candidate{Provider: provider})
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/chat", map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestChatCORSPreflight(t *testing.T) {
	provider := &fakeProvider{name: "fake", defaultModel: "fake-model"}
	modelRouter, err := router.New(router.Priority{}, router.Candidate{Provider: provider})
	if err != nil {
		t.Fatal(err)
	}
	server := New(modelRouter, WithCORS([]string{"http://example.com"}))
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodOptions, ts.URL+"/chat", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "http://example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://example.com" {
		t.Fatalf("allow-origin = %q", got)
	}
}

func TestChatTimeout(t *testing.T) {
	provider := &fakeProvider{
		name:         "fake",
		defaultModel: "fake-model",
		delay:        500 * time.Millisecond,
	}
	modelRouter, err := router.New(router.Priority{}, router.Candidate{Provider: provider})
	if err != nil {
		t.Fatal(err)
	}
	server := New(modelRouter, WithTimeout(50*time.Millisecond))
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/chat", map[string]any{
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestHealthEndpoint(t *testing.T) {
	provider := &fakeProvider{name: "fake", defaultModel: "fake-model"}
	ts := newTestServer(t, router.Candidate{Provider: provider})
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Fatalf("body = %q", string(body))
	}
}
