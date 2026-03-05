// Package agent provides the core agent functionality for GoClaw.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/zeroclaw-labs/goclaw/pkg/tools"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

// DefaultSystemPromptBuilder is the default system prompt builder.
type DefaultSystemPromptBuilder struct{}

// NewDefaultSystemPromptBuilder creates a new DefaultSystemPromptBuilder.
func NewDefaultSystemPromptBuilder() *DefaultSystemPromptBuilder {
	return &DefaultSystemPromptBuilder{}
}

// Build constructs a system prompt.
func (b *DefaultSystemPromptBuilder) Build(context, message string) string {
	prompt := fmt.Sprintf(`You are GoClaw, an AI assistant. Follow these guidelines:

1. Be helpful, honest, and harmless.
2. Use the provided tools to answer questions and complete tasks.
3. Format responses clearly and concisely.
4. If you don't know the answer, say so.
5. When user asks to send something to their email or mailbox, first use the memory_recall tool to search for "email" to find their email address.

Context:
%s

User message:
%s`, context, message)
	return prompt
}

// DefaultToolDispatcher is the default tool dispatcher.
type DefaultToolDispatcher struct{}

// NewDefaultToolDispatcher creates a new DefaultToolDispatcher.
func NewDefaultToolDispatcher() *DefaultToolDispatcher {
	return &DefaultToolDispatcher{}
}

// ExecuteTools executes multiple tool calls.
func (d *DefaultToolDispatcher) ExecuteTools(ctx context.Context, toolCalls []types.ToolCall, tools []tools.Tool) ([]ToolExecutionResult, error) {
	results := make([]ToolExecutionResult, len(toolCalls))

	for i, call := range toolCalls {
		result, err := d.executeTool(ctx, call, tools)
		if err != nil {
			return nil, fmt.Errorf("failed to execute tool %s: %w", call.Name, err)
		}
		results[i] = result
	}

	return results, nil
}

// executeTool executes a single tool call.
func (d *DefaultToolDispatcher) executeTool(ctx context.Context, call types.ToolCall, toolList []tools.Tool) (ToolExecutionResult, error) {
	// Find the tool
	var foundTool tools.Tool
	found := false
	for _, t := range toolList {
		if t.Name() == call.Name {
			foundTool = t
			found = true
			break
		}
	}

	if !found {
		return ToolExecutionResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("Unknown tool: %s", call.Name),
			Success:    false,
		}, nil
	}

	// Execute the tool
	var args map[string]interface{}
	log.Printf("Raw arguments: %s", string(call.Arguments))
	if err := json.Unmarshal(call.Arguments, &args); err != nil {
		log.Printf("Failed to unmarshal arguments: %v", err)
		return ToolExecutionResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("Error parsing arguments: %v", err),
			Success:    false,
			Error:      err.Error(),
		}, nil
	}
	log.Printf("Parsed arguments: %v", args)
	log.Printf("Executing tool: %s", call.Name)
	result, err := foundTool.Execute(ctx, args)
	if err != nil {
		log.Printf("Error executing tool: %v", err)
		return ToolExecutionResult{
			ToolCallID: call.ID,
			Output:     fmt.Sprintf("Error executing %s: %v", call.Name, err),
			Success:    false,
			Error:      err.Error(),
		}, nil
	}
	log.Printf("Tool execution result: success=%v, output=%s", result.Success, result.Output)

	// Scrub credentials from output
	output := scrubCredentials(result.Output)

	return ToolExecutionResult{
		ToolCallID: call.ID,
		Output:     output,
		Success:    result.Success,
		Error:      result.Error,
	}, nil
}

// scrubCredentials removes sensitive information from tool output.
func scrubCredentials(input string) string {
	// TODO: Implement credential scrubbing
	return input
}

// DefaultMemoryLoader is the default memory loader.
type DefaultMemoryLoader struct{}

// NewDefaultMemoryLoader creates a new DefaultMemoryLoader.
func NewDefaultMemoryLoader() *DefaultMemoryLoader {
	return &DefaultMemoryLoader{}
}

// LoadMemory loads relevant memory based on a query.
func (l *DefaultMemoryLoader) LoadMemory(ctx context.Context, memory Memory, query string) (string, error) {
	// Retrieve relevant memory entries
	entries, err := memory.Recall(ctx, query, 5, nil)
	if err != nil {
		return "", fmt.Errorf("failed to recall memory: %w", err)
	}

	if len(entries) == 0 {
		return "No relevant memory found", nil
	}

	// Build memory context
	var context string
	for _, entry := range entries {
		if score := entry.Score; score != nil {
			context += fmt.Sprintf("- %.2f: %s\n", *score, entry.Content)
		} else {
			context += fmt.Sprintf("- %s\n", entry.Content)
		}
	}

	return context, nil
}

// DefaultObserver is the default observer.
type DefaultObserver struct{}

// NewDefaultObserver creates a new DefaultObserver.
func NewDefaultObserver() *DefaultObserver {
	return &DefaultObserver{}
}

// RecordEvent records an agent event.
func (o *DefaultObserver) RecordEvent(event *ObserverEvent) {
	// TODO: Implement event recording
}

// StartTrace starts a new trace.
func (o *DefaultObserver) StartTrace(name string) Trace {
	return &DefaultTrace{}
}

// DefaultTrace is the default trace implementation.
type DefaultTrace struct{}

// End ends the trace.
func (t *DefaultTrace) End() {
	// TODO: Implement trace ending
}

// AddSpan adds a span to the trace.
func (t *DefaultTrace) AddSpan(name string, attributes map[string]interface{}) Span {
	return &DefaultSpan{}
}

// DefaultSpan is the default span implementation.
type DefaultSpan struct{}

// End ends the span.
func (s *DefaultSpan) End() {
	// TODO: Implement span ending
}

// SetAttribute sets an attribute on the span.
func (s *DefaultSpan) SetAttribute(key string, value interface{}) {
	// TODO: Implement attribute setting
}
