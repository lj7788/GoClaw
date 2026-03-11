package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type OpenAIProvider struct {
	baseURL           string
	apiKey            string
	maxTokensOverride *int
	httpClient        *http.Client
}

type OpenAIMessage struct {
	Role             string           `json:"role"`
	Content          *string          `json:"content,omitempty"`
	ToolCallID       *string          `json:"tool_call_id,omitempty"`
	ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`
	ReasoningContent *string          `json:"reasoning_content,omitempty"`
}

type OpenAIToolCall struct {
	ID       *string            `json:"id,omitempty"`
	Type     *string            `json:"type,omitempty"`
	Function OpenAIFunctionCall `json:"function"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type OpenAIChatRequest struct {
	Model       string           `json:"model"`
	Messages    []OpenAIMessage  `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   *int             `json:"max_tokens,omitempty"`
	Tools       []OpenAIToolSpec `json:"tools,omitempty"`
	ToolChoice  *string          `json:"tool_choice,omitempty"`
}

type OpenAIToolSpec struct {
	Type     string             `json:"type"`
	Function OpenAIFunctionSpec `json:"function"`
}

type OpenAIFunctionSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type OpenAIChatResponse struct {
	Choices []OpenAIChoice   `json:"choices"`
	Usage   *OpenAIUsageInfo `json:"usage,omitempty"`
}

type OpenAIChoice struct {
	Message OpenAIResponseMessage `json:"message"`
}

type OpenAIResponseMessage struct {
	Content          *string          `json:"content,omitempty"`
	ReasoningContent *string          `json:"reasoning_content,omitempty"`
	ToolCalls        []OpenAIToolCall `json:"tool_calls,omitempty"`
}

type OpenAIUsageInfo struct {
	PromptTokens     *int64 `json:"prompt_tokens,omitempty"`
	CompletionTokens *int64 `json:"completion_tokens,omitempty"`
}

func (m OpenAIResponseMessage) effectiveContent() string {
	content := ""
	if m.Content != nil && *m.Content != "" {
		content = *m.Content
	} else if m.ReasoningContent != nil {
		content = *m.ReasoningContent
	}
	
	// Remove <think>...</think> tags (reasoning/thinking blocks from models like MiniMax)
	return removeThinkTags(content)
}

func removeThinkTags(text string) string {
	result := ""
	rest := text
	for {
		start := strings.Index(rest, "<think>")
		if start == -1 {
			result += rest
			break
		}
		result += rest[:start]
		rest = rest[start+len("<think>"):]
		end := strings.Index(rest, "</think>")
		if end == -1 {
			break
		}
		rest = rest[end+len("</think>"):]
	}
	return strings.TrimSpace(result)
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return NewOpenAIProviderWithBaseURL("", apiKey)
}

func NewOpenAIProviderWithBaseURL(baseURL, apiKey string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &OpenAIProvider{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 300 * time.Second}, // 5 minutes timeout
	}
}

// NewCustomProvider creates a provider for custom OpenAI-compatible API endpoints
func NewCustomProvider(baseURL, apiKey string) *OpenAIProvider {
	return NewOpenAIProviderWithBaseURL(baseURL, apiKey)
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Capabilities() types.ProviderCapabilities {
	return types.ProviderCapabilities{
		NativeToolCalling: true,
		Vision:            false,
	}
}

func (p *OpenAIProvider) ConvertTools(tools []*types.ToolSpec) *ConvertToolsResult {
	if len(tools) == 0 {
		return &ConvertToolsResult{
			Type:         "openai",
			ToolsPayload: json.RawMessage("[]"),
		}
	}

	openAITools := make([]OpenAIToolSpec, len(tools))
	for i, tool := range tools {
		openAITools[i] = OpenAIToolSpec{
			Type: "function",
			Function: OpenAIFunctionSpec{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}

	toolsPayload, _ := json.Marshal(openAITools)
	return &ConvertToolsResult{
		Type:         "openai",
		ToolsPayload: toolsPayload,
	}
}

func (p *OpenAIProvider) SimpleChat(ctx context.Context, message, model string, temperature float64) (string, error) {
	return p.ChatWithSystem(ctx, "", message, model, temperature)
}

func (p *OpenAIProvider) ChatWithSystem(ctx context.Context, systemPrompt, message, model string, temperature float64) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not set. Set OPENAI_API_KEY or edit config.toml")
	}

	messages := make([]OpenAIMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: &systemPrompt,
		})
	}
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: &message,
	})

	req := OpenAIChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return "", err
	}

	chatResp, ok := resp.(OpenAIChatResponse)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return chatResp.Choices[0].Message.effectiveContent(), nil
}

func (p *OpenAIProvider) ChatWithHistory(ctx context.Context, messages []types.ChatMessage, model string, temperature float64) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not set. Set OPENAI_API_KEY or edit config.toml")
	}

	nativeMessages := p.convertMessages(messages)

	req := OpenAIChatRequest{
		Model:       model,
		Messages:    nativeMessages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return "", err
	}

	chatResp, ok := resp.(OpenAIChatResponse)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return chatResp.Choices[0].Message.effectiveContent(), nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, request *ChatRequest, model string, temperature float64) (*types.ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not set. Set OPENAI_API_KEY or edit config.toml")
	}

	tools := p.convertToolSpecs(request.Tools)
	nativeMessages := p.convertMessages(request.Messages)

	toolChoice := "auto"
	req := OpenAIChatRequest{
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

	chatResp, ok := resp.(OpenAIChatResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return p.parseResponse(chatResp), nil
}

func (p *OpenAIProvider) ChatWithTools(ctx context.Context, messages []types.ChatMessage, tools []json.RawMessage, model string, temperature float64) (*types.ChatResponse, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not set. Set OPENAI_API_KEY or edit config.toml")
	}

	openAITools := make([]OpenAIToolSpec, 0, len(tools))
	for _, tool := range tools {
		var spec OpenAIToolSpec
		if err := json.Unmarshal(tool, &spec); err != nil {
			return nil, fmt.Errorf("invalid OpenAI tool specification: %w", err)
		}
		if spec.Type != "function" {
			return nil, fmt.Errorf("invalid OpenAI tool specification: unsupported tool type '%s', expected 'function'", spec.Type)
		}
		openAITools = append(openAITools, spec)
	}

	nativeMessages := p.convertMessages(messages)

	toolChoice := "auto"
	req := OpenAIChatRequest{
		Model:       model,
		Messages:    nativeMessages,
		Temperature: temperature,
		MaxTokens:   p.maxTokensOverride,
		ToolChoice:  &toolChoice,
		Tools:       openAITools,
	}

	resp, err := p.doRequest(ctx, "/chat/completions", req)
	if err != nil {
		return nil, err
	}

	chatResp, ok := resp.(OpenAIChatResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return p.parseResponse(chatResp), nil
}

func (p *OpenAIProvider) SupportsNativeTools() bool {
	return true
}

func (p *OpenAIProvider) SupportsVision() bool {
	return false
}

func (p *OpenAIProvider) SupportsStreaming() bool {
	return false
}

func (p *OpenAIProvider) StreamChatWithSystem(ctx context.Context, systemPrompt, message, model string, temperature float64, options types.StreamOptions) (<-chan types.StreamChunk, error) {
	ch := make(chan types.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (p *OpenAIProvider) StreamChatWithHistory(ctx context.Context, messages []types.ChatMessage, model string, temperature float64, options types.StreamOptions) (<-chan types.StreamChunk, error) {
	ch := make(chan types.StreamChunk, 1)
	close(ch)
	return ch, nil
}

func (p *OpenAIProvider) Warmup(ctx context.Context) error {
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

func (p *OpenAIProvider) doRequest(ctx context.Context, path string, req interface{}) (interface{}, error) {
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
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result OpenAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (p *OpenAIProvider) convertToolSpecs(tools []*types.ToolSpec) []OpenAIToolSpec {
	if len(tools) == 0 {
		return nil
	}

	result := make([]OpenAIToolSpec, len(tools))
	for i, tool := range tools {
		result[i] = OpenAIToolSpec{
			Type: "function",
			Function: OpenAIFunctionSpec{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}
	return result
}

func (p *OpenAIProvider) convertMessages(messages []types.ChatMessage) []OpenAIMessage {
	result := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		result[i] = p.convertMessage(msg)
	}
	return result
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (p *OpenAIProvider) convertMessage(msg types.ChatMessage) OpenAIMessage {
			if msg.Role == "assistant" {
			log.Printf("Converting assistant message, content: %s", msg.Content)
			var toolCallsData map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Content), &toolCallsData); err == nil {
				log.Printf("Parsed toolCallsData: %+v", toolCallsData)
				if tc, ok := toolCallsData["tool_calls"]; ok {
					log.Printf("Found tool_calls: %+v", tc)
					if tcArr, ok := tc.([]interface{}); ok {
						toolCalls := make([]OpenAIToolCall, 0, len(tcArr))
						for _, tcItem := range tcArr {
							if tcMap, ok := tcItem.(map[string]interface{}); ok {
								id, _ := tcMap["id"].(string)
								log.Printf("Tool call ID from map: '%s'", id)
								var name, args string
								if funcObj, ok := tcMap["function"].(map[string]interface{}); ok {
									name, _ = funcObj["name"].(string)
									args, _ = funcObj["arguments"].(string)
								} else {
									name, _ = tcMap["name"].(string)
									args, _ = tcMap["arguments"].(string)
								}
								log.Printf("Creating tool call with ID: '%s', Name: '%s'", id, name)
								// 确保ID不为空
								if id == "" {
									log.Printf("WARNING: Empty tool call ID found, generating a new one")
									id = fmt.Sprintf("call_%d", time.Now().UnixNano())
								}
								toolCalls = append(toolCalls, OpenAIToolCall{
									ID:   &id,
									Type: strPtr("function"),
									Function: OpenAIFunctionCall{
										Name:      name,
										Arguments: args,
									},
								})
							}
						}
					var content *string
					if c, ok := toolCallsData["content"].(string); ok {
						content = &c
					}
					var reasoningContent *string
					if rc, ok := toolCallsData["reasoning_content"].(string); ok {
						reasoningContent = &rc
					}

					log.Printf("Returning assistant message with %d tool calls", len(toolCalls))
					return OpenAIMessage{
						Role:             "assistant",
						Content:          content,
						ToolCalls:        toolCalls,
						ReasoningContent: reasoningContent,
					}
				}
			}
		} else {
			log.Printf("Failed to parse assistant message content: %v", err)
		}
	}

	if msg.Role == "tool" {
		log.Printf("Processing tool message, content length: %d", len(msg.Content))
		// 直接解析消息内容 JSON
		var toolData map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Content), &toolData); err == nil {
			var toolCallID, content string
			if tcID, ok := toolData["tool_call_id"].(string); ok {
				toolCallID = tcID
				log.Printf("Found tool_call_id: '%s'", toolCallID)
			} else {
				log.Printf("ERROR: tool_call_id not found in message content! Content: %s", truncateString(msg.Content, 200))
				// 尝试从原始消息中提取
				if len(msg.Content) > 0 {
					// 从消息内容中提取 tool_call_id
					var tempToolData map[string]interface{}
					if err := json.Unmarshal([]byte(msg.Content), &tempToolData); err == nil {
						if tcID, ok := tempToolData["tool_call_id"].(string); ok {
							toolCallID = tcID
							log.Printf("Successfully extracted tool_call_id from content: '%s'", toolCallID)
						}
					}
				}
			}
			
			if c, ok := toolData["content"].(string); ok {
				content = c
			}
			
			// 确保 toolCallID 不为空
			if toolCallID == "" {
				log.Printf("ERROR: Cannot create OpenAIMessage with empty tool_call_id! Message content: %s", truncateString(msg.Content, 200))
				// 返回一个没有 tool_call_id 的消息，让上游处理
				return OpenAIMessage{
					Role:    "tool",
					Content: &content,
				}
			}
			
			log.Printf("Creating OpenAIMessage with tool_call_id: '%s'", toolCallID)
			return OpenAIMessage{
				Role:       "tool",
				Content:    &content,
				ToolCallID: &toolCallID,
			}
		} else {
			log.Printf("ERROR: Failed to parse tool message content as JSON: %v", err)
		}
	}

	return OpenAIMessage{
		Role:    string(msg.Role),
		Content: &msg.Content,
	}
}

func (p *OpenAIProvider) parseResponse(resp OpenAIChatResponse) *types.ChatResponse {
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

func strPtr(s string) *string {
	return &s
}

func uintPtr(v uint64) *uint64 {
	return &v
}
