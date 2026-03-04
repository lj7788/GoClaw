// Package agent provides the core agent functionality for GoClaw.
package agent

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/zeroclaw-labs/goclaw/pkg/providers"
	"github.com/zeroclaw-labs/goclaw/pkg/skills"
	"github.com/zeroclaw-labs/goclaw/pkg/tools"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

// Agent represents an AI agent with core functionality.
type Agent struct {
	provider             providers.Provider
	tools                []tools.Tool
	toolSpecs            []*types.ToolSpec
	memory               Memory
	observer             Observer
	promptBuilder        SystemPromptBuilder
	toolDispatcher       ToolDispatcher
	memoryLoader         MemoryLoader
	config               AgentConfig
	modelName            string
	temperature          float64
	workspaceDir         string
	identityConfig       IdentityConfig
	skills               []*skills.Skill
	skillLoader          *skills.SkillLoader
	skillsPromptMode     SkillsPromptInjectionMode
	autoSave             bool
	history              []types.ConversationMessage
	classificationConfig QueryClassificationConfig
	availableHints       []string
	routeModelByHint     map[string]string
	mu                   sync.RWMutex
}

// AgentBuilder is used to construct an Agent instance.
type AgentBuilder struct {
	agent *Agent
}

// NewAgentBuilder creates a new AgentBuilder.
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		agent: &Agent{
			routeModelByHint: make(map[string]string),
		},
	}
}

// WithProvider sets the LLM provider for the agent.
func (b *AgentBuilder) WithProvider(provider providers.Provider) *AgentBuilder {
	b.agent.provider = provider
	return b
}

// WithTools sets the tools available to the agent.
func (b *AgentBuilder) WithTools(tools []tools.Tool) *AgentBuilder {
	b.agent.tools = tools
	// Generate tool specs
	toolSpecs := make([]*types.ToolSpec, len(tools))
	for i, tool := range tools {
		spec := tool.Spec()
		toolSpecs[i] = &types.ToolSpec{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  spec.Parameters,
		}
	}
	b.agent.toolSpecs = toolSpecs
	return b
}

// WithMemory sets the memory backend for the agent.
func (b *AgentBuilder) WithMemory(memory Memory) *AgentBuilder {
	b.agent.memory = memory
	return b
}

// WithObserver sets the observer for monitoring agent activity.
func (b *AgentBuilder) WithObserver(observer Observer) *AgentBuilder {
	b.agent.observer = observer
	return b
}

// WithPromptBuilder sets the prompt builder for constructing system prompts.
func (b *AgentBuilder) WithPromptBuilder(promptBuilder SystemPromptBuilder) *AgentBuilder {
	b.agent.promptBuilder = promptBuilder
	return b
}

// WithToolDispatcher sets the tool dispatcher for executing tools.
func (b *AgentBuilder) WithToolDispatcher(toolDispatcher ToolDispatcher) *AgentBuilder {
	b.agent.toolDispatcher = toolDispatcher
	return b
}

// WithMemoryLoader sets the memory loader for loading relevant memories.
func (b *AgentBuilder) WithMemoryLoader(memoryLoader MemoryLoader) *AgentBuilder {
	b.agent.memoryLoader = memoryLoader
	return b
}

// WithConfig sets the agent configuration.
func (b *AgentBuilder) WithConfig(config AgentConfig) *AgentBuilder {
	b.agent.config = config
	return b
}

// WithModelName sets the default model name for the agent.
func (b *AgentBuilder) WithModelName(modelName string) *AgentBuilder {
	b.agent.modelName = modelName
	return b
}

// WithTemperature sets the temperature for LLM responses.
func (b *AgentBuilder) WithTemperature(temperature float64) *AgentBuilder {
	b.agent.temperature = temperature
	return b
}

// WithWorkspaceDir sets the workspace directory for the agent.
func (b *AgentBuilder) WithWorkspaceDir(workspaceDir string) *AgentBuilder {
	b.agent.workspaceDir = workspaceDir
	return b
}

// WithIdentityConfig sets the identity configuration for the agent.
func (b *AgentBuilder) WithIdentityConfig(identityConfig IdentityConfig) *AgentBuilder {
	b.agent.identityConfig = identityConfig
	return b
}

// WithSkills sets the skills available to the agent.
func (b *AgentBuilder) WithSkills(skills []*skills.Skill) *AgentBuilder {
	b.agent.skills = skills
	return b
}

// WithSkillLoader sets the skill loader for automatic skill loading.
// When set, skills will be automatically loaded from the skills directory
// and registered as tools - enabling zero-code skill extension.
func (b *AgentBuilder) WithSkillLoader(skillLoader *skills.SkillLoader) *AgentBuilder {
	b.agent.skillLoader = skillLoader
	return b
}

// WithSkillsPromptMode sets the skills prompt injection mode.
func (b *AgentBuilder) WithSkillsPromptMode(mode SkillsPromptInjectionMode) *AgentBuilder {
	b.agent.skillsPromptMode = mode
	return b
}

// WithAutoSave enables or disables auto-save functionality.
func (b *AgentBuilder) WithAutoSave(autoSave bool) *AgentBuilder {
	b.agent.autoSave = autoSave
	return b
}

// WithClassificationConfig sets the query classification configuration.
func (b *AgentBuilder) WithClassificationConfig(config QueryClassificationConfig) *AgentBuilder {
	b.agent.classificationConfig = config
	return b
}

// WithAvailableHints sets the available hints for the agent.
func (b *AgentBuilder) WithAvailableHints(hints []string) *AgentBuilder {
	b.agent.availableHints = hints
	return b
}

// WithRouteModelByHint sets the model routing configuration.
func (b *AgentBuilder) WithRouteModelByHint(routes map[string]string) *AgentBuilder {
	for k, v := range routes {
		b.agent.routeModelByHint[k] = v
	}
	return b
}

// Build constructs and returns the Agent instance.
func (b *AgentBuilder) Build() (*Agent, error) {
	// Validate required fields
	if b.agent.provider == nil {
		return nil, fmt.Errorf("provider is required")
	}
	if b.agent.memory == nil {
		return nil, fmt.Errorf("memory is required")
	}
	if b.agent.promptBuilder == nil {
		b.agent.promptBuilder = NewDefaultSystemPromptBuilder()
	}
	if b.agent.toolDispatcher == nil {
		b.agent.toolDispatcher = NewDefaultToolDispatcher()
	}
	if b.agent.memoryLoader == nil {
		b.agent.memoryLoader = NewDefaultMemoryLoader()
	}
	if b.agent.modelName == "" {
		b.agent.modelName = "gpt-4o"
	}
	if b.agent.temperature == 0 {
		b.agent.temperature = 0.7
	}
	if b.agent.workspaceDir == "" {
		b.agent.workspaceDir = "."
	}

	// Auto-load skills from skill loader and register as tools
	if b.agent.skillLoader != nil {
		log.Printf("Loading skills from directory: %s", b.agent.skillLoader.GetSkillsDir())
		if err := b.agent.skillLoader.LoadSkills(); err != nil {
			return nil, fmt.Errorf("failed to load skills: %w", err)
		}
		loadedSkills := b.agent.skillLoader.ListSkills()
		log.Printf("Loaded %d skills", len(loadedSkills))
		for _, skill := range loadedSkills {
			log.Printf("  - %s (%d tools)", skill.Name, len(skill.Tools))
		}

		// Add skills to agent
		b.agent.skills = append(b.agent.skills, loadedSkills...)

		// Convert skill tools to agent tools and register
		skillTools := skills.ConvertSkillToolsToTools(loadedSkills, b.agent.skillLoader.GetSkillsDir())
		log.Printf("Converted %d skill tools to agent tools", len(skillTools))
		b.agent.tools = append(b.agent.tools, skillTools...)

		// Generate tool specs for skill tools
		skillToolSpecs := skills.ConvertSkillToolsToToolSpecs(loadedSkills)
		b.agent.toolSpecs = append(b.agent.toolSpecs, skillToolSpecs...)
	}

	return b.agent, nil
}

// ProcessMessage processes a user message and returns a response.
func (a *Agent) ProcessMessage(ctx context.Context, message string) (*types.ChatResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add user message to history
	a.history = append(a.history, types.ConversationMessage{
		Type: "chat",
		Chat: &types.ChatMessage{
			Role:    types.RoleUser,
			Content: message,
		},
	})

	// Build context
	contextStr, err := a.buildContext(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to build context: %w", err)
	}

	// Build prompt
	prompt := a.promptBuilder.Build(contextStr, message)

	// Call LLM
	response, err := a.provider.Chat(ctx, &providers.ChatRequest{
		Messages: []types.ChatMessage{
			{Role: types.RoleSystem, Content: prompt},
			{Role: types.RoleUser, Content: message},
		},
		Tools: a.toolSpecs,
	}, a.modelName, a.temperature)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	// Process response
	if response.HasToolCalls() {
		// Execute tools
		toolResults, err := a.toolDispatcher.ExecuteTools(ctx, response.ToolCalls, a.tools)
		if err != nil {
			return nil, fmt.Errorf("failed to execute tools: %w", err)
		}

		// Add tool results to history
		for _, result := range toolResults {
			a.history = append(a.history, types.ConversationMessage{
				Type: "tool_results",
				ToolResults: []types.ToolResultMessage{
					{
						ToolCallID: result.ToolCallID,
						Content:    result.Output,
					},
				},
			})
		}
		// Call LLM again with tool results - build full conversation context
		responseText := ""
		if response.Text != nil {
			responseText = *response.Text
		}

		// Build messages for second call
		// Use simple text format for tool results - works with any provider
		messages := []types.ChatMessage{
			{Role: types.RoleSystem, Content: prompt},
			{Role: types.RoleUser, Content: message},
		}

		// Add assistant message with tool call info
		if len(response.ToolCalls) > 0 {
			messages = append(messages, types.ChatMessage{
				Role:    types.RoleAssistant,
				Content: responseText,
			})

			// Add tool results as a user message (simple text format)
			var toolResultsText string
			for _, result := range toolResults {
				toolResultsText += result.Output + "\n\n"
			}

			messages = append(messages, types.ChatMessage{
				Role:    types.RoleUser,
				Content: "Tool results:\n" + toolResultsText,
			})
		}

		response, err = a.provider.Chat(ctx, &providers.ChatRequest{
			Messages: messages,
			Tools:    a.toolSpecs,
		}, a.modelName, a.temperature)
		if err != nil {
			return nil, fmt.Errorf("failed to call LLM with tool results: %w", err)
		}
	}

	// Add assistant response to history
	a.history = append(a.history, types.ConversationMessage{
		Type: "chat",
		Chat: &types.ChatMessage{
			Role:    types.RoleAssistant,
			Content: response.TextOrEmpty(),
		},
	})

	return response, nil
}

// buildContext constructs the context for the agent.
func (a *Agent) buildContext(ctx context.Context, message string) (string, error) {
	// Load relevant memory
	memoryContext, err := a.memoryLoader.LoadMemory(ctx, a.memory, message)
	if err != nil {
		return "", fmt.Errorf("failed to load memory: %w", err)
	}

	// Build hardware context if needed
	hardwareContext := ""
	// TODO: Implement hardware context building

	// Combine contexts
	context := fmt.Sprintf("[Memory context]\n%s\n\n[Hardware context]\n%s", memoryContext, hardwareContext)
	return context, nil
}

// Tools returns the list of tools available to the agent.
func (a *Agent) Tools() []tools.Tool {
	return a.tools
}

// ToolSpecs returns the tool specifications for the agent.
func (a *Agent) ToolSpecs() []*types.ToolSpec {
	return a.toolSpecs
}

// History returns the conversation history.
func (a *Agent) History() []types.ConversationMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()
	// Return a copy to prevent modification
	history := make([]types.ConversationMessage, len(a.history))
	copy(history, a.history)
	return history
}

// ClearHistory clears the conversation history.
func (a *Agent) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.history = nil
}

// SaveMemory saves the current conversation to memory.
func (a *Agent) SaveMemory(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// TODO: Implement memory saving
	return nil
}

// LoadMemory loads conversation history from memory.
func (a *Agent) LoadMemory(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// TODO: Implement memory loading
	return nil
}

// TrimHistory trims the conversation history to prevent unbounded growth.
func (a *Agent) TrimHistory(maxHistory int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// TODO: Implement history trimming
}

// AutoCompactHistory automatically compacts the conversation history.
func (a *Agent) AutoCompactHistory(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// TODO: Implement history compaction
	return nil
}
