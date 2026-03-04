package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

const (
	defaultBailianBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
)

type BailianProvider struct {
	baseURL           string
	apiKey            string
	maxTokensOverride *int
	httpClient        *http.Client
}

type BailianMessage struct {
	Role             string           `json:"role"`
	Content          *string          `json:"content,omitempty"`
	ToolCallID       *string          `json:"tool_call_id,omitempty"`
	ToolCalls        []BailianToolCall `json:"tool_calls,omitempty"`
	ReasoningContent *string          `json:"reasoning_content,omitempty"`
}

type BailianToolCall struct {
	ID       *string            `json:"id,omitempty"`
	Type     *string            `json:"type,omitempty"`
	Function BailianFunctionCall `json:"function"`
}

type BailianFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type BailianChatRequest struct {
	Model       string              `json:"model"`
	Messages    []BailianMessage     `json:"messages"`
	Temperature float64              `json:"temperature"`
	MaxTokens   *int                 `json:"max_tokens,omitempty"`
	Tools       []BailianToolSpec    `json:"tools,omitempty"`
	ToolChoice  *string              `json:"tool_choice,omitempty"`
}

type BailianToolSpec struct {
	Type     string               `json:"type"`
	Function BailianFunctionSpec   `json:"function"`
}

type BailianFunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type BailianChatResponse struct {
	Choices []BailianChoice   `json:"choices"`
	Usage   *BailianUsageInfo `json:"usage,omitempty"`
}

type BailianChoice struct {
	Message BailianResponseMessage `json:"message"`
}

type BailianResponseMessage struct {
	Content          *string          `json:"content,omitempty"`
	ReasoningContent *string          `json:"reasoning_content,omitempty"`
	ToolCalls        []BailianToolCall `json:"tool_calls,omitempty"`
}

type BailianUsageInfo struct {
	PromptTokens     *int64 `json:"prompt_tokens,omitempty"`
	CompletionTokens *int64 `json:"completion_tokens,omitempty"`
}

func (m BailianResponseMessage) effectiveContent() string {
	if m.Content != nil && *m.Content != "" {
		return *m.Content
	}
	if m.ReasoningContent != nil {
		return *m.ReasoningContent
	}
	return ""
}

func NewBailianProvider(apiKey string) *BailianProvider {
	return NewBailianProviderWithBaseURL("", apiKey)
}

func NewBailianProviderWithBaseURL(baseURL, apiKey string) *BailianProvider {
	if baseURL == "" {
		baseURL = defaultBailianBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &BailianProvider{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *BailianProvider) Name() string {
	return "bailian"
}

func (p *BailianProvider) Capabilities() types.ProviderCapabilities {
	return types.ProviderCapabilities{
		NativeToolCalling: true,
		Vision:            false,
	}
}

func (p *BailianProvider) ConvertTools(tools []*types.ToolSpec) *ConvertToolsResult {
	if len(tools) == 0 {
		return &ConvertToolsResult{
			Type:         "openai",
			ToolsPayload: json.RawMessage("[]"),
		}
	}

	bailianTools := make([]BailianToolSpec, len(tools))
	for i, tool := range tools {
		bailianTools[i] = BailianToolSpec{
			Type: "function",
			Function: BailianFunctionSpec{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}

	toolsPayload, _ := json.Marshal(bailianTools)
	return &ConvertToolsResult{
		Type:         "openai",
		ToolsPayload: toolsPayload,
	}
}

func (p *BailianProvider) SimpleChat(ctx context.Context, message, model string, temperature float64) (string, error) {
	return p.ChatWithSystem(ctx, "", message, model, temperature)
}

func (p *BailianProvider) ChatWithSystem(ctx context.Context, systemPrompt, message, model string, temperature float64) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("Bailian API key not set. Set BAILIAN_API_KEY or edit config.toml")
	}

	messages := make([]BailianMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, BailianMessage{
			Role:    "system",
			Content: &systemPrompt,
		})
	}
	messages = append(messages, BailianMessage{
		Role:    "user",
		Content: &message,
	})

	req := BailianChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return "", err
	}

	chatResp, ok := resp.(BailianChatResponse)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Bailian")
	}

	return chatResp.Choices[0].Message.effectiveContent(), nil
}

func (p *BailianProvider) ChatWithHistory(ctx context.Context, messages []types.ChatMessage, model string, temperature float64) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("Bailian API key not set. Set BAILIAN_API_KEY or edit config.toml")
	}

	nativeMessages := p.convertMessages(messages)

	req := BailianChatRequest{
		Model:       model,
		Messages:    nativeMessages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return "", err
	}

	chatResp, ok := resp.(BailianChatResponse)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Bailian")
	}

	return chatResp.Choices[0].Message.effectiveContent(), nil
}

func (p *BailianProvider) Chat(ctx context.Context, request *ChatRequest, model string, temperature float64) (*types.ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Bailian API key not set. Set BAILIAN_API_KEY or edit config.toml")
	}

	tools := p.convertToolSpecs(request.Tools)
	nativeMessages := p.convertMessages(request.Messages)

	toolChoice := "auto"
	req := BailianChatRequest{
		Model:       model,
		Messages:    nativeMessages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
		ToolChoice:  &toolChoice,
		Tools:       tools,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return nil, err
	}

	chatResp, ok := resp.(BailianChatResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Bailian")
	}

	return p.parseResponse(chatResp), nil
}

func (p *BailianProvider) ChatWithTools(ctx context.Context, messages []types.ChatMessage, tools []json.RawMessage, model string, temperature float64) (*types.ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("Bailian API key not set. Set BAILIAN_API_KEY or edit config.toml")
	}

	bailianTools := make([]BailianToolSpec, 0, len(tools))
	for _, tool := range tools {
		var spec BailianToolSpec
		if err := json.Unmarshal(tool, &spec); err != nil {
			return nil, fmt.Errorf("invalid Bailian tool specification: %w", err)
		}
		if spec.Type != "function" {
			return nil, fmt.Errorf("invalid Bailian tool specification: unsupported tool type '%s', expected 'function'", spec.Type)
		}
		bailianTools = append(bailianTools, spec)
	}

	nativeMessages := p.convertMessages(messages)

	toolChoice := "auto"
	req := BailianChatRequest{
		Model:       model,
		Messages:    nativeMessages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
		ToolChoice:  &toolChoice,
		Tools:       bailianTools,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return nil, err
	}

	chatResp, ok := resp.(BailianChatResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Bailian")
	}

	return p.parseResponse(chatResp), nil
}

func (p *BailianProvider) SupportsNativeTools() bool {
	return true
}

func (p *BailianProvider) SupportsVision() bool {
	return false
}

func (p *BailianProvider) SupportsStreaming() bool {
	return false
}

func (p *BailianProvider) StreamChatWithSystem(ctx context.Context, systemPrompt, message, model string, temperature float64, options types.StreamOptions) (<-chan types.StreamChunk, error) {
	ch := make(chan types.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (p *BailianProvider) StreamChatWithHistory(ctx context.Context, messages []types.ChatMessage, model string, temperature float64, options types.StreamOptions) (<-chan types.StreamChunk, error) {
	ch := make(chan types.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (p *BailianProvider) Warmup(ctx context.Context) error {
	if p.apiKey == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("warmup failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (p *BailianProvider) doRequest(ctx context.Context, path string, req interface{}) (interface{}, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bailian API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result BailianChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (p *BailianProvider) convertToolSpecs(tools []*types.ToolSpec) []BailianToolSpec {
	if len(tools) == 0 {
		return nil
	}

	result := make([]BailianToolSpec, len(tools))
	for i, tool := range tools {
		result[i] = BailianToolSpec{
			Type: "function",
			Function: BailianFunctionSpec{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return result
}

func (p *BailianProvider) convertMessages(messages []types.ChatMessage) []BailianMessage {
	result := make([]BailianMessage, len(messages))
	for i, msg := range messages {
		result[i] = p.convertMessage(msg)
	}
	return result
}

func (p *BailianProvider) convertMessage(msg types.ChatMessage) BailianMessage {
	if msg.Role == "assistant" {
		if data, err := json.Marshal(msg.Content); err == nil {
			var toolCallsData map[string]interface{}
			if json.Unmarshal(data, &toolCallsData); toolCallsData != nil {
				if tc, ok := toolCallsData["tool_calls"]; ok {
					if tcArr, ok := tc.([]interface{}); ok {
						toolCalls := make([]BailianToolCall, 0, len(tcArr))
						for _, tcItem := range tcArr {
							if tcMap, ok := tcItem.(map[string]interface{}); ok {
								id, _ := tcMap["id"].(string)
								name, _ := tcMap["name"].(string)
								args, _ := tcMap["arguments"].(string)
								toolCalls = append(toolCalls, BailianToolCall{
									ID:   &id,
									Type: strPtr("function"),
									Function: BailianFunctionCall{
										Name:      name,
										Arguments: args,
									},
								})
							}
						}

						content, _ := toolCallsData["content"].(string)
						var reasoningContent *string
						if rc, ok := toolCallsData["reasoning_content"].(string); ok {
							reasoningContent = &rc
						}

						return BailianMessage{
							Role:             "assistant",
							Content:          &content,
							ToolCalls:        toolCalls,
							ReasoningContent: reasoningContent,
						}
					}
				}
			}
		}
	}

	if msg.Role == "tool" {
		if data, err := json.Marshal(msg.Content); err == nil {
			var toolData map[string]interface{}
			if json.Unmarshal(data, &toolData); toolData != nil {
				var toolCallID, content string
				if tcID, ok := toolData["tool_call_id"].(string); ok {
					toolCallID = tcID
				}
				if c, ok := toolData["content"].(string); ok {
					content = c
				}
				return BailianMessage{
					Role:       "tool",
					Content:    &content,
					ToolCallID: &toolCallID,
				}
			}
		}
	}

	return BailianMessage{
		Role:    string(msg.Role),
		Content: &msg.Content,
	}
}

func (p *BailianProvider) parseResponse(resp BailianChatResponse) *types.ChatResponse {
	msg := resp.Choices[0].Message

	text := msg.effectiveContent()
	reasoningContent := msg.ReasoningContent

	toolCalls := make([]types.ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		id := ""
		if tc.ID != nil {
			id = *tc.ID
		}
		toolCalls = append(toolCalls, types.ToolCall{
			ID:        id,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}

	var usage *types.TokenUsage
	if resp.Usage != nil {
		usage = &types.TokenUsage{}
		if resp.Usage.PromptTokens != nil {
			usage.InputTokens = uintPtr(uint64(*resp.Usage.PromptTokens))
		}
		if resp.Usage.CompletionTokens != nil {
			usage.OutputTokens = uintPtr(uint64(*resp.Usage.CompletionTokens))
		}
	}

	return &types.ChatResponse{
		Text:             &text,
		ToolCalls:        toolCalls,
		Usage:            usage,
		ReasoningContent: reasoningContent,
	}
}
