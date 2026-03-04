package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/tools"
	"github.com/zeroclaw-labs/goclaw/pkg/types"
)

// SkillToolExecutor executes skill tools based on their kind (shell/http/script).
// This enables zero-code skill loading similar to zeroclaw-fix-cn.
type SkillToolExecutor struct {
	skill    *Skill
	tool     SkillTool
	skillDir string
}

// NewSkillToolExecutor creates a new SkillToolExecutor.
func NewSkillToolExecutor(skill *Skill, tool SkillTool, skillDir string) *SkillToolExecutor {
	return &SkillToolExecutor{
		skill:    skill,
		tool:     tool,
		skillDir: skillDir,
	}
}

// Name returns the tool name in format "skillname:toolname".
func (e *SkillToolExecutor) Name() string {
	return e.skill.Name + ":" + e.tool.Name
}

// Description returns the tool description.
func (e *SkillToolExecutor) Description() string {
	return e.tool.Description
}

// ParametersSchema returns the JSON schema for tool parameters.
func (e *SkillToolExecutor) ParametersSchema() json.RawMessage {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{},
	}

	// Add parameters from tool definition if available
	if len(e.tool.Parameters) > 0 {
		properties := make(map[string]interface{})
		required := []string{}
		for _, param := range e.tool.Parameters {
			properties[param.Name] = map[string]interface{}{
				"type":        param.Type,
				"description": param.Description,
			}
			if param.Required {
				required = append(required, param.Name)
			}
		}
		schema["properties"] = properties
		if len(required) > 0 {
			schema["required"] = required
		}
		data, _ := json.Marshal(schema)
		return json.RawMessage(data)
	}

	// Fallback to default schema
	switch e.tool.Kind {
	case "shell":
		schema["properties"] = map[string]interface{}{
			"command": map[string]string{
				"type":        "string",
				"description": "Shell command to execute",
			},
		}
		schema["required"] = []string{"command"}
	case "http":
		schema["properties"] = map[string]interface{}{
			"method": map[string]string{
				"type":        "string",
				"description": "HTTP method (GET, POST, PUT, DELETE)",
				"default":     "GET",
			},
			"path": map[string]string{
				"type":        "string",
				"description": "URL path to append to base URL",
			},
			"body": map[string]string{
				"type":        "string",
				"description": "Request body (for POST/PUT)",
			},
		}
	case "script":
		schema["properties"] = map[string]interface{}{
			"script": map[string]string{
				"type":        "string",
				"description": "Script content to execute",
			},
		}
		schema["required"] = []string{"script"}
	}

	data, _ := json.Marshal(schema)
	return json.RawMessage(data)
}

// Spec returns the tool specification.
func (e *SkillToolExecutor) Spec() tools.ToolSpec {
	return tools.ToolSpec{
		Name:        e.Name(),
		Description: e.Description(),
		Parameters:  e.ParametersSchema(),
	}
}

// Execute executes the skill tool based on its kind.
func (e *SkillToolExecutor) Execute(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	switch e.tool.Kind {
	case "shell":
		return e.executeShell(ctx, args)
	case "http":
		return e.executeHTTP(ctx, args)
	case "script":
		return e.executeScript(ctx, args)
	default:
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   fmt.Sprintf("unknown tool kind: %s", e.tool.Kind),
		}, nil
	}
}

// executeShell executes a shell command.
func (e *SkillToolExecutor) executeShell(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	command := e.tool.Command

	// If args contain a command, use it instead (for backward compatibility)
	if cmd, ok := args["command"].(string); ok && cmd != "" {
		command = cmd
	}

	if command == "" {
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   "no command specified",
		}, nil
	}

	// Support template substitution from args
	// Add default values for missing parameters
	for _, param := range e.tool.Parameters {
		if _, exists := args[param.Name]; !exists && param.Default != "" {
			args[param.Name] = param.Default
		}
	}
	for k, v := range args {
		placeholder := "{{" + k + "}}"
		command = strings.ReplaceAll(command, placeholder, fmt.Sprintf("%v", v))
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   "empty command",
		}, nil
	}

	// Set working directory to skill directory
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if e.skillDir != "" {
		cmd.Dir = e.skillDir
	}
	
	output, err := cmd.CombinedOutput()
	fmt.Printf("[DEBUG] Command output: %s\n", string(output))
	fmt.Printf("[DEBUG] Command error: %v\n", err)

	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Output:  string(output),
			Error:   err.Error(),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// executeHTTP executes an HTTP request.
func (e *SkillToolExecutor) executeHTTP(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	baseURL := e.tool.Command

	// Get HTTP method
	method := "GET"
	if m, ok := args["method"].(string); ok {
		method = m
	}

	// Build URL
	path := ""
	if p, ok := args["path"].(string); ok {
		path = p
	}

	// Support template substitution
	for k, v := range args {
		placeholder := "{{" + k + "}}"
		baseURL = strings.ReplaceAll(baseURL, placeholder, fmt.Sprintf("%v", v))
		path = strings.ReplaceAll(path, placeholder, fmt.Sprintf("%v", v))
	}

	// Merge base URL and path
	url := baseURL
	if path != "" {
		if !strings.HasSuffix(url, "/") && !strings.HasPrefix(path, "/") {
			url += "/"
		}
		url += path
	}

	// Get body
	var body io.Reader
	if b, ok := args["body"].(string); ok && b != "" {
		body = strings.NewReader(b)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   err.Error(),
		}, nil
	}

	// Add headers from tool args
	for k, v := range e.tool.Args {
		if strings.HasPrefix(k, "header:") {
			headerName := strings.TrimPrefix(k, "header:")
			req.Header.Add(headerName, v)
		}
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   err.Error(),
		}, nil
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	return &tools.ToolResult{
		Success: success,
		Output:  string(respBody),
		Error:   fmt.Sprintf("HTTP %d", resp.StatusCode),
	}, nil
}

// executeScript executes a script (uses shell to run).
func (e *SkillToolExecutor) executeScript(ctx context.Context, args map[string]interface{}) (*tools.ToolResult, error) {
	script := e.tool.Command

	// If args contain script content, use it
	if s, ok := args["script"].(string); ok && s != "" {
		script = s
	}

	// Support template substitution
	for k, v := range args {
		placeholder := "{{" + k + "}}"
		script = strings.ReplaceAll(script, placeholder, fmt.Sprintf("%v", v))
	}

	if script == "" {
		return &tools.ToolResult{
			Success: false,
			Output:  "",
			Error:   "no script specified",
		}, nil
	}

	// Determine interpreter from tool args
	interpreter := "sh"
	if ints, ok := e.tool.Args["interpreter"]; ok {
		interpreter = ints
	}

	var cmd *exec.Cmd
	if interpreter == "bash" || interpreter == "sh" || interpreter == "zsh" {
		cmd = exec.CommandContext(ctx, interpreter, "-c", script)
	} else if interpreter == "python" || interpreter == "python3" {
		cmd = exec.CommandContext(ctx, interpreter, "-c", script)
	} else if interpreter == "node" {
		cmd = exec.CommandContext(ctx, interpreter, "-e", script)
	} else {
		// Try as shell script
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", script)
	}

	output, err := cmd.CombinedOutput()

	if err != nil {
		return &tools.ToolResult{
			Success: false,
			Output:  string(output),
			Error:   err.Error(),
		}, nil
	}

	return &tools.ToolResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// ConvertSkillToolsToTools converts skill tools to tools.Tool implementations.
func ConvertSkillToolsToTools(skills []*Skill, skillsDir string) []tools.Tool {
	var result []tools.Tool

	for _, skill := range skills {
		for _, tool := range skill.Tools {
			result = append(result, NewSkillToolExecutor(skill, tool, skillsDir))
		}
	}

	return result
}

// ConvertSkillToolsToToolSpecs converts skill tools to types.ToolSpec.
func ConvertSkillToolsToToolSpecs(skills []*Skill) []*types.ToolSpec {
	var result []*types.ToolSpec

	for _, skill := range skills {
		for _, tool := range skill.Tools {
			executor := NewSkillToolExecutor(skill, tool, "")
			spec := executor.Spec()
			result = append(result, &types.ToolSpec{
				Name:        spec.Name,
				Description: spec.Description,
				Parameters:  spec.Parameters,
			})
		}
	}

	return result
}

// SkillToPrompt converts a skill to its prompt representation.
func SkillToPrompt(skill *Skill, workspaceDir string) string {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("## Skill: %s\n", skill.Name))
	buf.WriteString(fmt.Sprintf("Description: %s\n\n", skill.Description))

	if len(skill.Prompts) > 0 {
		buf.WriteString("### Instructions\n")
		for _, prompt := range skill.Prompts {
			buf.WriteString(prompt)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	if len(skill.Tools) > 0 {
		buf.WriteString("### Tools\n")
		for _, tool := range skill.Tools {
			buf.WriteString(fmt.Sprintf("- `%s:%s` - %s (kind: %s)\n",
				skill.Name, tool.Name, tool.Description, tool.Kind))
		}
	}

	return buf.String()
}
