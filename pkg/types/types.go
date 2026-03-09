// Package types provides core type definitions for the GoClaw agent runtime.
package types

import (
	"context"
	"encoding/json"
	"time"
)

// MessageRole represents the role of a message sender
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// ChatMessage represents a single message in a conversation
type ChatMessage struct {
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

// NewChatMessage creates a new chat message
func NewChatMessage(role MessageRole, content string) ChatMessage {
	return ChatMessage{
		Role:    role,
		Content: content,
	}
}

// System creates a system message
func System(content string) ChatMessage {
	return ChatMessage{Role: RoleSystem, Content: content}
}

// User creates a user message
func User(content string) ChatMessage {
	return ChatMessage{Role: RoleUser, Content: content}
}

// Assistant creates an assistant message
func Assistant(content string) ChatMessage {
	return ChatMessage{Role: RoleAssistant, Content: content}
}

// Tool creates a tool result message
func Tool(content string) ChatMessage {
	return ChatMessage{Role: RoleTool, Content: content}
}

// ToolCall represents a tool call requested by the LLM
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ConversationMessage represents a message in a multi-turn conversation
type ConversationMessage struct {
	Type        string              `json:"type"` // "chat", "tool_results"
	Chat        *ChatMessage        `json:"chat,omitempty"`
	ToolResults []ToolResultMessage `json:"tool_results,omitempty"`
}

const MessageTypeChat = "chat"

// ToolResultMessage represents a tool result to feed back to the LLM
type ToolResultMessage struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

// TokenUsage represents raw token counts from a single LLM API response
type TokenUsage struct {
	InputTokens  *uint64 `json:"input_tokens,omitempty"`
	OutputTokens *uint64 `json:"output_tokens,omitempty"`
}

// ChatResponse represents an LLM response that may contain text, tool calls, or both
type ChatResponse struct {
	Text             *string     `json:"text,omitempty"`
	ToolCalls        []ToolCall  `json:"tool_calls,omitempty"`
	Usage            *TokenUsage `json:"usage,omitempty"`
	ReasoningContent *string     `json:"reasoning_content,omitempty"` // For thinking models
}

// HasToolCalls returns true when the LLM wants to invoke at least one tool
func (r *ChatResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// TextOrEmpty returns the text content or an empty string
func (r *ChatResponse) TextOrEmpty() string {
	if r.Text != nil {
		return *r.Text
	}
	return ""
}

// StreamChunk represents a chunk of content from a streaming response
type StreamChunk struct {
	Delta      string `json:"delta"`
	IsFinal    bool   `json:"is_final"`
	TokenCount int    `json:"token_count"`
}

// NewStreamChunk creates a new non-final chunk
func NewStreamChunk(delta string) StreamChunk {
	return StreamChunk{
		Delta:   delta,
		IsFinal: false,
	}
}

// FinalChunk creates a final chunk
func FinalChunk() StreamChunk {
	return StreamChunk{
		Delta:   "",
		IsFinal: true,
	}
}

// ErrorChunk creates an error chunk
func ErrorChunk(message string) StreamChunk {
	return StreamChunk{
		Delta:   message,
		IsFinal: true,
	}
}

// StreamOptions represents options for streaming chat requests
type StreamOptions struct {
	Enabled     bool `json:"enabled"`
	CountTokens bool `json:"count_tokens"`
}

// NewStreamOptions creates new streaming options
func NewStreamOptions(enabled bool) StreamOptions {
	return StreamOptions{
		Enabled:     enabled,
		CountTokens: false,
	}
}

// WithTokenCount enables token counting
func (o StreamOptions) WithTokenCount() StreamOptions {
	o.CountTokens = true
	return o
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// NewToolResult creates a successful tool result
func NewToolResult(output string) ToolResult {
	return ToolResult{
		Success: true,
		Output:  output,
	}
}

// NewToolError creates a failed tool result
func NewToolError(err string) ToolResult {
	return ToolResult{
		Success: false,
		Error:   err,
	}
}

// ToolSpec represents a tool description for the LLM
type ToolSpec struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ProviderCapabilities describes what features a provider supports
type ProviderCapabilities struct {
	NativeToolCalling bool `json:"native_tool_calling"`
	Vision            bool `json:"vision"`
}

// DefaultCapabilities returns minimal capabilities
func DefaultCapabilities() ProviderCapabilities {
	return ProviderCapabilities{
		NativeToolCalling: false,
		Vision:            false,
	}
}

// Stream represents a stream of chat chunks
type Stream interface {
	Next(ctx context.Context) (StreamChunk, error)
	Close() error
}

// ChannelMessage represents a message received from or sent to a channel
type ChannelMessage struct {
	ID          string `json:"id"`
	Sender      string `json:"sender"`
	ReplyTarget string `json:"reply_target"`
	Content     string `json:"content"`
	Channel     string `json:"channel"`
	Timestamp   uint64 `json:"timestamp"`
	ThreadTS    string `json:"thread_ts,omitempty"`
	MessageID   string `json:"message_id,omitempty"`
}

// SendMessage represents a message to send through a channel
type SendMessage struct {
	Content   string  `json:"content"`
	Recipient string  `json:"recipient"`
	Subject   *string `json:"subject,omitempty"`
	ThreadTS  *string `json:"thread_ts,omitempty"`
}

// NewSendMessage creates a new message with content and recipient
func NewSendMessage(content, recipient string) *SendMessage {
	return &SendMessage{
		Content:   content,
		Recipient: recipient,
	}
}

// WithSubject creates a new message with subject
func (m *SendMessage) WithSubject(subject string) *SendMessage {
	m.Subject = &subject
	return m
}

// InThread sets the thread identifier for threaded replies
func (m *SendMessage) InThread(threadTS string) *SendMessage {
	m.ThreadTS = &threadTS
	return m
}

// MemoryCategory represents memory categories for organization
type MemoryCategory string

const (
	MemoryCategoryCore         MemoryCategory = "core"
	MemoryCategoryDaily        MemoryCategory = "daily"
	MemoryCategoryConversation MemoryCategory = "conversation"
)

// CustomMemoryCategory creates a custom memory category
func CustomMemoryCategory(name string) MemoryCategory {
	return MemoryCategory(name)
}

// String returns the string representation of the category
func (c MemoryCategory) String() string {
	return string(c)
}

// MemoryEntry represents a single memory entry
type MemoryEntry struct {
	ID        string          `json:"id"`
	Key       string          `json:"key"`
	Content   string          `json:"content"`
	Category  MemoryCategory  `json:"category"`
	Timestamp string          `json:"timestamp"`
	SessionID *string         `json:"session_id,omitempty"`
	Score     *float64        `json:"score,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// ProgressEvent represents a progress update during execution
type ProgressEvent struct {
	Type      string      `json:"type"` // "tool_start", "tool_progress", "tool_complete", "error"
	Tool      string      `json:"tool,omitempty"`
	Step      int         `json:"step,omitempty"`
	Total     int         `json:"total,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// NewProgressEvent creates a new progress event
func NewProgressEvent(eventType string) ProgressEvent {
	return ProgressEvent{
		Type:      eventType,
		Timestamp: time.Now(),
	}
}

// WithTool sets the tool name
func (e ProgressEvent) WithTool(name string) ProgressEvent {
	e.Tool = name
	return e
}

// WithStep sets the current step
func (e ProgressEvent) WithStep(step, total int) ProgressEvent {
	e.Step = step
	e.Total = total
	return e
}

// WithMessage sets the message
func (e ProgressEvent) WithMessage(msg string) ProgressEvent {
	e.Message = msg
	return e
}

// OpenAIChatRequest represents an OpenAI-compatible chat request
type OpenAIChatRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Tools       json.RawMessage `json:"tools,omitempty"`
}

// OpenAIMessage represents an OpenAI message
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// OpenAIChatResponse represents an OpenAI-compatible chat response
type OpenAIChatResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int            `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
}

// OpenAIChoice represents a chat completion choice
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIChunk represents a streaming chat chunk
type OpenAIChunk struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int                 `json:"created"`
	Model   string              `json:"model"`
	Choices []OpenAIChunkChoice `json:"choices"`
}

// OpenAIChunkChoice represents a streaming chunk choice
type OpenAIChunkChoice struct {
	Index        int           `json:"index"`
	Delta        OpenAIMessage `json:"delta"`
	FinishReason string        `json:"finish_reason"`
}
