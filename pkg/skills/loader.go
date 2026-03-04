package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Skill struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Version     string                 `json:"version"`
	Author      string                 `json:"author,omitempty"`
	Commands    []SkillCommand         `json:"commands"`
	Metadata    map[string]interface{} `json:"metadata"`
	Tools       []SkillTool            `json:"tools,omitempty"`
	Prompts     []string               `json:"prompts,omitempty"`
}

type SkillCommand struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Aliases     []string         `json:"aliases,omitempty"`
	Command     string           `json:"command,omitempty"`
	Kind        string           `json:"kind,omitempty"`
	Parameters  []SkillParameter `json:"parameters,omitempty"`
}

type SkillParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// SkillTool defines a tool that can be executed by a skill.
// Similar to zeroclaw-fix-cn's SkillTool, supporting shell/http/script types.
type SkillTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Kind        string            `json:"kind"` // "shell", "http", "script"
	Command     string            `json:"command"`
	Args        map[string]string `json:"args,omitempty"`
	Parameters  []SkillParameter  `json:"parameters,omitempty"`
}

type SkillLoader struct {
	mu        sync.RWMutex
	skills    map[string]*Skill
	skillsDir string
}

func NewSkillLoader(skillsDir string) *SkillLoader {
	return &SkillLoader{
		skills:    make(map[string]*Skill),
		skillsDir: skillsDir,
	}
}

// GetSkillsDir returns the skills directory path.
func (l *SkillLoader) GetSkillsDir() string {
	return l.skillsDir
}

// LoadSkills loads skills from both skill.json and SKILL.toml/SKILL.md files.
// This enables zero-code skill loading similar to zeroclaw-fix-cn.
func (l *SkillLoader) LoadSkills() error {
	if l.skillsDir == "" {
		return nil
	}

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(l.skillsDir, entry.Name())

		// Try skill.json first (legacy format)
		jsonPath := filepath.Join(skillPath, "skill.json")
		if data, err := os.ReadFile(jsonPath); err == nil {
			var skill Skill
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&skill); err != nil {
				log.Printf("Failed to decode skill.json for %s: %v", entry.Name(), err)
				// Try SKILL.toml if skill.json fails
			} else {
				// Use directory name as skill name
				skill.Name = entry.Name()
				
				// Convert commands to tools if tools are not provided
				if len(skill.Tools) == 0 && len(skill.Commands) > 0 {
					skill.Tools = make([]SkillTool, 0, len(skill.Commands))
					for _, cmd := range skill.Commands {
						command := fmt.Sprintf("./%s.sh", cmd.Name)
						if cmd.Command != "" {
							command = cmd.Command
						}
						tool := SkillTool{
							Name:        cmd.Name,
							Description: cmd.Description,
							Kind:        "shell",
							Command:      command,
							Parameters:  cmd.Parameters,
						}
						skill.Tools = append(skill.Tools, tool)
					}
				}
				log.Printf("Loaded skill %s from skill.json with %d tools", skill.Name, len(skill.Tools))
				l.mu.Lock()
				l.skills[skill.Name] = &skill
				l.mu.Unlock()
			}
			continue
		}

		// Try SKILL.toml (new format similar to zeroclaw-fix-cn)
		tomlPath := filepath.Join(skillPath, "SKILL.toml")
		if data, err := os.ReadFile(tomlPath); err == nil {
			var manifest SkillManifest
			if err := json.Unmarshal(data, &manifest); err == nil {
				skill := manifest.toSkill()
				skill.Name = entry.Name()
				l.mu.Lock()
				l.skills[skill.Name] = skill
				l.mu.Unlock()
				continue
			}
		}

		// Try SKILL.md (simple markdown format)
		mdPath := filepath.Join(skillPath, "SKILL.md")
		if data, err := os.ReadFile(mdPath); err == nil {
			skill := loadSkillFromMD(entry.Name(), string(data))
			l.mu.Lock()
			l.skills[skill.Name] = skill
			l.mu.Unlock()
		}
	}

	return nil
}

// SkillManifest represents a skill loaded from SKILL.toml
type SkillManifest struct {
	Skill   SkillMeta   `json:"skill"`
	Commands []SkillCommand `json:"commands,omitempty"`
	Tools   []SkillTool   `json:"tools,omitempty"`
	Prompts []string    `json:"prompts,omitempty"`
}

type SkillMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      *string  `json:"author,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func (m *SkillManifest) toSkill() *Skill {
	version := m.Skill.Version
	if version == "" {
		version = "0.1.0"
	}

	// Convert commands to tools if tools are not provided
	tools := m.Tools
	if len(tools) == 0 && len(m.Commands) > 0 {
		tools = make([]SkillTool, 0, len(m.Commands))
		for _, cmd := range m.Commands {
			tool := SkillTool{
				Name:        cmd.Name,
				Description: cmd.Description,
				Kind:        "shell",
				Command:      fmt.Sprintf("./%s.sh", cmd.Name),
				Parameters:  cmd.Parameters,
			}
			tools = append(tools, tool)
		}
	}

	return &Skill{
		Name:        m.Skill.Name,
		Description: m.Skill.Description,
		Version:     version,
		Commands:    m.Commands,
		Tools:       tools,
		Prompts:     m.Prompts,
	}
}

func loadSkillFromMD(name string, content string) *Skill {
	// Extract description from first non-heading, non-empty line
	desc := "No description"
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > 0 && trimmed[0] == '#' {
			continue
		}
		desc = trimmed
		break
	}

	// Parse commands from SKILL.md
	var commands []SkillCommand
	var tools []SkillTool

	// Look for Commands section
	lines := strings.Split(content, "\n")
	inCommandsSection := false
	var currentCommand *SkillCommand
	var currentCommandIndex = -1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're in the Commands section
		if strings.HasPrefix(trimmed, "## Commands") || strings.HasPrefix(trimmed, "### Commands") {
			inCommandsSection = true
			continue
		}
		
		// Exit commands section when we hit another major section
		if inCommandsSection && strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "## Commands") {
			inCommandsSection = false
			continue
		}

		// Parse command definitions
		if inCommandsSection {
			// Look for command definition: **Command Name** (`command_name`): description
			if strings.HasPrefix(trimmed, "- **") && (strings.Contains(trimmed, "** (") || strings.Contains(trimmed, "** (`")) {
				// Extract command name and description
				parts := strings.Split(trimmed, "**")
				if len(parts) >= 3 {
					// The command name is in parentheses in parts[2]
					// parts[2] is like " (analyze): description" or " (`analyze`): description"
					cmdName := ""
					if cmdNameStart := strings.Index(parts[2], "("); cmdNameStart != -1 {
						remaining := parts[2][cmdNameStart:]
						if cmdNameEnd := strings.Index(remaining, ")"); cmdNameEnd != -1 {
							cmdName = strings.TrimSpace(remaining[1:cmdNameEnd])
						}
					}
					
					cmdDesc := ""
					if len(parts) >= 3 {
						// Find the position of "):" or "`):" to extract description
						descStart := strings.Index(parts[2], "):")
						if descStart == -1 {
							descStart = strings.Index(parts[2], "`):")
						}
						if descStart != -1 {
							cmdDesc = strings.TrimSpace(parts[2][descStart+2:])
						}
					}
					
					if cmdName != "" {
						currentCommand = &SkillCommand{
							Name:        cmdName,
							Description: cmdDesc,
						}
						commands = append(commands, *currentCommand)
						currentCommandIndex = len(commands) - 1
					}
				}
			} else if currentCommand != nil && currentCommandIndex >= 0 {
				// Check if it's a parameters line (supports 2, 4, or 6 space indentation)
				for strings.HasPrefix(trimmed, "  - ") {
					trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "  - "))
				}
				
				// Skip Parameters header
				if strings.HasPrefix(trimmed, "Parameters:") || strings.HasPrefix(trimmed, "- Parameters:") {
					continue
				}
				
				// Now check if it's a parameter line (should have colon)
				if strings.Contains(trimmed, ":") {
					paramParts := strings.SplitN(trimmed, ":", 2)
					
					if len(paramParts) == 2 {
						paramName := strings.TrimSpace(paramParts[0])
						paramDesc := strings.TrimSpace(paramParts[1])
						
						// Remove leading "- " from parameter name if present
						if strings.HasPrefix(paramName, "- ") {
							paramName = strings.TrimSpace(paramName[2:])
						}
						
						// Remove (required) and (optional) from parameter name
						isRequired := false
						if strings.Contains(paramName, "(required)") || strings.Contains(paramName, "（必需）") {
							isRequired = true
							paramName = strings.TrimSpace(strings.Split(paramName, " (")[0])
							paramName = strings.TrimSpace(strings.Split(paramName, "（")[0])
						} else if strings.Contains(paramName, "(optional)") || strings.Contains(paramName, "（可选）") {
							isRequired = false
							paramName = strings.TrimSpace(strings.Split(paramName, " (")[0])
							paramName = strings.TrimSpace(strings.Split(paramName, "（")[0])
						}
						
						newParam := SkillParameter{
							Name:        paramName,
							Type:        "string",
							Description: paramDesc,
							Required:    isRequired,
						}
						currentCommand.Parameters = append(currentCommand.Parameters, newParam)
						
						// Update the command in the commands slice
						commands[currentCommandIndex] = *currentCommand
					}
				}
			}
		}
	}

	// Create tools from commands
	for _, cmd := range commands {
		tool := SkillTool{
			Name:        cmd.Name,
			Description: cmd.Description,
			Kind:        "shell",
			Command:      fmt.Sprintf("./%s.sh", cmd.Name),
			Parameters:  cmd.Parameters,
		}
		tools = append(tools, tool)
	}

	return &Skill{
		Name:        name,
		Description: desc,
		Version:     "0.1.0",
		Commands:    commands,
		Tools:       tools,
		Prompts:     []string{content},
	}
}

func (l *SkillLoader) GetSkill(name string) (*Skill, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	skill, exists := l.skills[name]
	return skill, exists
}

func (l *SkillLoader) ListSkills() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	skills := make([]*Skill, 0, len(l.skills))
	for _, skill := range l.skills {
		skills = append(skills, skill)
	}

	return skills
}

// GetAllTools returns all tools from all loaded skills.
func (l *SkillLoader) GetAllTools() []SkillTool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var tools []SkillTool
	for _, skill := range l.skills {
		tools = append(tools, skill.Tools...)
	}
	return tools
}

func (l *SkillLoader) AddSkill(skill *Skill) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.skills[skill.Name] = skill

	skillDir := filepath.Join(l.skillsDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	data, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill: %w", err)
	}

	skillPath := filepath.Join(skillDir, "skill.json")
	if err := os.WriteFile(skillPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

func (l *SkillLoader) RemoveSkill(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.skills[name]; !exists {
		return fmt.Errorf("skill not found: %s", name)
	}

	delete(l.skills, name)

	skillDir := filepath.Join(l.skillsDir, name)
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove skill directory: %w", err)
	}

	return nil
}

type SkillExecutor struct {
	mu       sync.RWMutex
	handlers map[string]SkillHandler
}

type SkillHandler func(ctx context.Context, skill *Skill, command string, args map[string]interface{}) (string, error)

func NewSkillExecutor() *SkillExecutor {
	return &SkillExecutor{
		handlers: make(map[string]SkillHandler),
	}
}

func (e *SkillExecutor) RegisterHandler(skillName string, handler SkillHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[skillName] = handler
}

func (e *SkillExecutor) Execute(ctx context.Context, skill *Skill, command string, args map[string]interface{}) (string, error) {
	e.mu.RLock()
	handler, exists := e.handlers[skill.Name]
	e.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("no handler registered for skill: %s", skill.Name)
	}

	return handler(ctx, skill, command, args)
}

type SkillMetadata struct {
	Author     string    `json:"author"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	License    string    `json:"license"`
	Repository string    `json:"repository"`
	Tags       []string  `json:"tags"`
}

func (s *Skill) GetCommand(name string) *SkillCommand {
	for _, cmd := range s.Commands {
		if cmd.Name == name {
			return &cmd
		}
		for _, alias := range cmd.Aliases {
			if alias == name {
				return &cmd
			}
		}
	}
	return nil
}

func (s *Skill) ValidateCommand(name string, args map[string]interface{}) error {
	cmd := s.GetCommand(name)
	if cmd == nil {
		return fmt.Errorf("command not found: %s", name)
	}

	for _, param := range cmd.Parameters {
		if param.Required {
			if _, exists := args[param.Name]; !exists {
				return fmt.Errorf("required parameter missing: %s", param.Name)
			}
		}
	}

	return nil
}

// GetTool returns a specific tool from the skill by name.
func (s *Skill) GetTool(name string) *SkillTool {
	for _, tool := range s.Tools {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}
