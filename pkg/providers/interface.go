package providers

import (
	"context"
	"encoding/json"

	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

type ChatRequest struct {
	Messages    []types.ChatMessage
	Tools       []*types.ToolSpec
	Model       string
	Temperature float64
}

type ChatMessage struct {
	Role    string
	Content string
}

type ConvertToolsResult struct {
	Type         string
	ToolsPayload json.RawMessage
	Instructions string
}

func BuildToolInstructionsText(tools []*types.ToolSpec) string {
	return ""
}

type Provider interface {
	Name() string
	Chat(ctx context.Context, request *ChatRequest, model string, temperature float64) (*types.ChatResponse, error)
	Capabilities() types.ProviderCapabilities
}

type Model struct {
	ID   string
	Name string
}

type ProviderFactory func(config map[string]string) (Provider, error)

var providers = make(map[string]ProviderFactory)

func Register(name string, factory ProviderFactory) {
	providers[name] = factory
}

func NewProvider(name string, config map[string]string) (Provider, error) {
	factory, ok := providers[name]
	if !ok {
		return nil, nil
	}
	return factory(config)
}

func SupportedProviders() []string {
	return []string{
		"openai",
		"anthropic",
		"gemini",
		"glm",
		"ollama",
		"bailian",
	}
}

func init() {
	Register("openai", func(config map[string]string) (Provider, error) {
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, nil
		}
		return NewOpenAIProvider(apiKey), nil
	})
	Register("anthropic", func(config map[string]string) (Provider, error) {
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, nil
		}
		return NewAnthropicProvider(apiKey), nil
	})
	Register("gemini", func(config map[string]string) (Provider, error) {
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, nil
		}
		return NewGeminiProvider(apiKey), nil
	})
	Register("glm", func(config map[string]string) (Provider, error) {
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, nil
		}
		return NewGLMProvider(apiKey), nil
	})
	Register("ollama", func(config map[string]string) (Provider, error) {
		return NewOllamaProvider(), nil
	})
	Register("bailian", func(config map[string]string) (Provider, error) {
		apiKey := config["api_key"]
		if apiKey == "" {
			return nil, nil
		}
		return NewBailianProvider(apiKey), nil
	})
}
