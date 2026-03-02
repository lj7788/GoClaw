// Package agent provides the core agent functionality for GoClaw.
package agent

import (
	"context"

	"github.com/zeroclaw-labs/goclaw/pkg/types"
	"github.com/zeroclaw-labs/goclaw/pkg/tools"
)

// Memory represents the memory backend for the agent.
type Memory interface {
	// Recall retrieves relevant memory entries based on a query.
	Recall(ctx context.Context, query string, limit int, category *string) ([]MemoryEntry, error)

	// Store saves a memory entry.
	Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error

	// Get retrieves a specific memory entry by key.
	Get(ctx context.Context, key string) (*MemoryEntry, error)

	// Search searches memory entries based on a query.
	Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error)

	// Forget removes a memory entry.
	Forget(ctx context.Context, key string) error

	// Clear removes all memory entries.
	Clear(ctx context.Context) error
}

// MemoryEntry represents a single memory entry.
type MemoryEntry struct {
	Key      string            `json:"key"`
	Content  string            `json:"content"`
	Category *string           `json:"category,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Score    *float64          `json:"score,omitempty"`
}

// Observer monitors agent activity.
type Observer interface {
	// RecordEvent records an agent event.
	RecordEvent(event *ObserverEvent)

	// StartTrace starts a new trace.
	StartTrace(name string) Trace
}

// ObserverEvent represents an agent event.
type ObserverEvent struct {
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// Trace represents a trace of agent activity.
type Trace interface {
	// End ends the trace.
	End()

	// AddSpan adds a span to the trace.
	AddSpan(name string, attributes map[string]interface{}) Span
}

// Span represents a span within a trace.
type Span interface {
	// End ends the span.
	End()

	// SetAttribute sets an attribute on the span.
	SetAttribute(key string, value interface{})
}

// SystemPromptBuilder builds system prompts for the agent.
type SystemPromptBuilder interface {
	// Build constructs a system prompt.
	Build(context, message string) string
}

// ToolDispatcher dispatches tool calls.
type ToolDispatcher interface {
	// ExecuteTools executes multiple tool calls.
	ExecuteTools(ctx context.Context, toolCalls []types.ToolCall, tools []tools.Tool) ([]ToolExecutionResult, error)
}

// ToolExecutionResult represents the result of a tool execution.
type ToolExecutionResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// MemoryLoader loads relevant memory for the agent.
type MemoryLoader interface {
	// LoadMemory loads relevant memory based on a query.
	LoadMemory(ctx context.Context, memory Memory, query string) (string, error)
}

// Skill represents a custom skill that extends agent functionality.
type Skill interface {
	// Name returns the skill name.
	Name() string

	// Description returns the skill description.
	Description() string

	// Execute executes the skill.
	Execute(ctx context.Context, args map[string]interface{}) (interface{}, error)
}

// AgentConfig represents the agent configuration.
type AgentConfig struct {
	MaxToolIterations         int     `json:"max_tool_iterations"`
	MaxHistoryMessages        int     `json:"max_history_messages"`
	AutoSave                  bool    `json:"auto_save"`
	AutoCompact               bool    `json:"auto_compact"`
	MinRelevanceScore         float64 `json:"min_relevance_score"`
	WorkspaceDir              string  `json:"workspace_dir"`
	DefaultModel              string  `json:"default_model"`
	DefaultTemperature        float64 `json:"default_temperature"`
	SkillsPromptInjectionMode string  `json:"skills_prompt_injection_mode"`
}

// IdentityConfig represents the agent identity configuration.
type IdentityConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        string `json:"role"`
	Avatar      string `json:"avatar"`
}

// QueryClassificationConfig represents the query classification configuration.
type QueryClassificationConfig struct {
	Enabled bool    `json:"enabled"`
	Threshold float64 `json:"threshold"`
}

// SkillsPromptInjectionMode represents the mode for injecting skills into prompts.
type SkillsPromptInjectionMode string

const (
	// SkillsPromptModeDisabled disables skill injection.
	SkillsPromptModeDisabled SkillsPromptInjectionMode = "disabled"
	// SkillsPromptModeAll injects all skills.
	SkillsPromptModeAll SkillsPromptInjectionMode = "all"
	// SkillsPromptModeRelevant injects only relevant skills.
	SkillsPromptModeRelevant SkillsPromptInjectionMode = "relevant"
)
