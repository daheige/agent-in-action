package factory

import (
	"fmt"

	"agent-in-action/llm-gateway/internal/appconfig"
	"agent-in-action/llm-gateway/internal/llm"
	"agent-in-action/llm-gateway/internal/providers/claude"
	"agent-in-action/llm-gateway/internal/providers/gemini"
	"agent-in-action/llm-gateway/internal/providers/openai"
	"agent-in-action/llm-gateway/internal/router"
	"agent-in-action/llm-gateway/internal/transport"
)

func BuildRouter(config appconfig.Config) (*router.Router, error) {
	candidates, err := BuildAll(config)
	if err != nil {
		return nil, err
	}
	strategy, err := BuildStrategy(config.Strategy)
	if err != nil {
		return nil, err
	}
	return router.New(strategy, candidates...)
}

// BuildAll 按配置顺序构造 Provider，返回 slice 以保留 Priority 语义。
func BuildAll(config appconfig.Config) ([]router.Candidate, error) {
	httpClient := transport.NewClient()
	candidates := make([]router.Candidate, 0, len(config.Order))

	for _, name := range config.Order {
		providerConfig := config.Providers[name]
		var provider llm.Provider
		var err error

		switch name {
		case "deepseek", "doubao":
			provider, err = openai.New(openai.Config{
				Name:         providerConfig.Name,
				BaseURL:      providerConfig.BaseURL,
				APIKey:       providerConfig.APIKey,
				DefaultModel: providerConfig.Model,
				Capabilities: llm.Capability{Streaming: true},
				Client:       httpClient,
			})
		case "ollama":
			provider, err = openai.New(openai.Config{
				Name:         providerConfig.Name,
				BaseURL:      providerConfig.BaseURL,
				APIKey:       "ollama",
				DefaultModel: providerConfig.Model,
				Capabilities: llm.Capability{Streaming: true},
				Client:       httpClient,
			})
		case "claude":
			provider, err = claude.New(claude.Config{
				Name:         providerConfig.Name,
				BaseURL:      providerConfig.BaseURL,
				APIKey:       providerConfig.APIKey,
				DefaultModel: providerConfig.Model,
				Capabilities: llm.Capability{Streaming: true},
				Client:       httpClient,
			})
		case "gemini":
			provider, err = gemini.New(gemini.Config{
				Name:         providerConfig.Name,
				BaseURL:      providerConfig.BaseURL,
				APIKey:       providerConfig.APIKey,
				DefaultModel: providerConfig.Model,
				Capabilities: llm.Capability{Streaming: true},
				Client:       httpClient,
			})
		default:
			return nil, fmt.Errorf("不支持的 Provider %q", name)
		}
		if err != nil {
			return nil, fmt.Errorf("构建 %s Provider: %w", name, err)
		}
		candidates = append(candidates, router.Candidate{
			Provider:    provider,
			Pricing:     providerConfig.Pricing,
			LatencyHint: providerConfig.LatencyHint,
		})
	}
	return candidates, nil
}

func BuildStrategy(name string) (router.Strategy, error) {
	switch name {
	case "priority":
		return router.Priority{}, nil
	case "cheapest":
		return router.CheapestFirst{}, nil
	case "latency":
		return router.LowestLatency{}, nil
	default:
		return nil, fmt.Errorf("未知路由策略 %q", name)
	}
}
