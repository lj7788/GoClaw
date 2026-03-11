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
type DefaultSystemPromptBuilder struct {
	locale string
}

// NewDefaultSystemPromptBuilder creates a new DefaultSystemPromptBuilder.
func NewDefaultSystemPromptBuilder() *DefaultSystemPromptBuilder {
	return &DefaultSystemPromptBuilder{
		locale: "en-US",
	}
}

// WithLocale sets the locale for the prompt builder.
func (b *DefaultSystemPromptBuilder) WithLocale(locale string) *DefaultSystemPromptBuilder {
	b.locale = locale
	return b
}

// Build constructs a system prompt.
func (b *DefaultSystemPromptBuilder) Build(context, message string) string {
	var prompt string
	if b.locale == "zh-CN" || b.locale == "zh" {
		prompt = fmt.Sprintf(`你是一个名为 GoClaw 的 AI 助手。请遵循以下准则：

1. 乐于助人、诚实且无害。
2. 使用提供的工具回答问题和完成任务。
3. 清晰简洁地格式化响应。
4. 如果不知道答案，请如实告知。
5. 任务完成策略：
   - 仔细分析用户的完整请求，包括所有部分
   - 将多步骤任务分解为顺序操作
   - 完成请求的所有部分后再停止
   - 不要在只完成部分请求后停止
6. 定时任务创建（重要！）：
   - 当用户提到"定时"、"计划"、"每天"、"每周"、"每月"、"定期"、"重复"、"每隔X分钟"等词汇时，必须立即使用 cron_add 工具创建定时任务
   - cron_add 工具支持两种模式：
     * agent_task: 让 Agent 执行任务（推荐用于复杂任务，如"分析股票并发送到企业微信"）
     * command: 执行 Shell 命令（用于简单命令）
   - 使用示例：
     * 用户说"每10分钟分析股票并发送到企业微信" → cron_add(expression="*/10 * * * *", agent_task="分析股票爱尔眼科，把结果发到我的企业微信")
     * 用户说"每天早上9点提醒我开会" → cron_add(expression="0 9 * * *", agent_task="提醒用户开会")
   - Cron 表达式格式：分 时 日 月 周
   - 常用表达式：
     * "*/10 * * * *" = 每10分钟
     * "*/30 * * * *" = 每30分钟
     * "0 9 * * *" = 每天上午9:00
     * "0 16 * * 1-5" = 工作日每天下午16:00
     * "0 18 * * 5" = 每周五下午18:00
7. 仔细阅读每个工具的描述，了解具体的使用说明和要求。
8. 在完成用户请求的全部内容后立即停止。不要进行不必要的工具调用。

上下文：
%s

用户消息：
%s`, context, message)
	} else {
		prompt = fmt.Sprintf(`You are GoClaw, an AI assistant. Follow these guidelines:

1. Be helpful, honest, and harmless.
2. Use the provided tools to answer questions and complete tasks.
3. Format responses clearly and concisely.
4. If you don't know the answer, say so.
5. Task completion strategy:
   - Carefully analyze the user's complete request, including all parts
   - Break down multi-step tasks into sequential actions
   - Complete ALL parts of the request before stopping
   - Do not stop after completing only part of the request
6. Scheduled task creation (IMPORTANT!):
   - When users mention "schedule", "plan", "daily", "weekly", "monthly", "regular", "repeat", "every X minutes", etc., you MUST immediately use the cron_add tool to create a scheduled task
   - cron_add tool supports two modes:
     * agent_task: Let Agent execute tasks (recommended for complex tasks like "analyze stock and send to WeChat")
     * command: Execute Shell commands (for simple commands)
   - Usage examples:
     * User says "every 10 minutes analyze stock and send to WeChat" → cron_add(expression="*/10 * * * *", agent_task="Analyze stock Aier Eye Hospital and send results to my WeChat Work")
     * User says "remind me to meeting every day at 9am" → cron_add(expression="0 9 * * *", agent_task="Remind user about meeting")
   - Cron expression format: minute hour day month weekday
   - Common expressions:
     * "*/10 * * * *" = Every 10 minutes
     * "*/30 * * * *" = Every 30 minutes
     * "0 9 * * *" = Every day at 9:00 AM
     * "0 16 * * 1-5" = Every weekday at 4:00 PM
     * "0 18 * * 5" = Every Friday at 6:00 PM
7. Carefully read each tool's description for specific usage instructions and requirements.
8. Stop immediately after completing the ENTIRE user request. Do not make unnecessary tool calls.

Context:
%s

User message:
%s`, context, message)
	}
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
