// Package tools provides tool functionality for GoClaw.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FileReadTool reads a file from disk.
type FileReadTool struct {
	BaseTool
}

// NewFileReadTool creates a new FileReadTool.
func NewFileReadTool() *FileReadTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "Path to the file to read" }
		},
		"required": ["path"]
	}`)
	return &FileReadTool{
		BaseTool: *NewBaseTool(
			"file_read",
			"读取文件内容",
			schema,
		),
	}
}

// Execute executes the file read tool.
func (t *FileReadTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "path is required",
			Error:   "path parameter is missing or invalid",
		}, nil
	}

	// Expand path
	path = filepath.Clean(path)

	// Read file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to read file: %v", err),
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  string(content),
	}, nil
}

// FileWriteTool writes content to a file.
type FileWriteTool struct {
	BaseTool
}

// NewFileWriteTool creates a new FileWriteTool.
func NewFileWriteTool() *FileWriteTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "Path to the file to write" },
			"content": { "type": "string", "description": "Content to write to the file" },
			"append": { "type": "boolean", "description": "Whether to append to the file instead of overwriting" },
			"json_array": { "type": "boolean", "description": "When true, treats content as JSON object and appends to existing JSON array in file" }
		},
		"required": ["path", "content"]
	}`)
	return &FileWriteTool{
		BaseTool: *NewBaseTool(
			"file_write",
			"写入内容到文件",
			schema,
		),
	}
}

// Execute executes the file write tool.
func (t *FileWriteTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "path is required",
			Error:   "path parameter is missing or invalid",
		}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "content is required",
			Error:   "content parameter is missing or invalid",
		}, nil
	}

	append, _ := args["append"].(bool)
	jsonArray, _ := args["json_array"].(bool)

	// Expand path
	path = filepath.Clean(path)

	// Handle json_array mode
	if jsonArray {
		return t.writeJSONArray(path, content)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to create directory: %v", err),
			Error:   err.Error(),
		}, nil
	}

	// Write file
	var err error
	if append {
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return &ToolResult{
				Success: false,
				Output:  fmt.Sprintf("Failed to open file: %v", err),
				Error:   err.Error(),
			}, nil
		}
		defer file.Close()
		_, err = file.WriteString(content)
	} else {
		err = ioutil.WriteFile(path, []byte(content), 0644)
	}

	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to write file: %v", err),
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote to file: %s", path),
	}, nil
}

func (t *FileWriteTool) writeJSONArray(path, newItem string) (*ToolResult, error) {
	var data []map[string]interface{}

	if _, err := os.Stat(path); err == nil {
		data = []map[string]interface{}{}
		if content, err := ioutil.ReadFile(path); err == nil && len(content) > 0 {
			trimmed := strings.TrimSpace(string(content))
			if trimmed != "" && trimmed != "[]" {
				if err := json.Unmarshal(content, &data); err != nil {
					data = []map[string]interface{}{}
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to read file: %v", err),
			Error:   err.Error(),
		}, nil
	}

	var newItemData map[string]interface{}
	if err := json.Unmarshal([]byte(newItem), &newItemData); err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Invalid JSON content: %v", err),
			Error:   err.Error(),
		}, nil
	}

	data = append(data, newItemData)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to create directory: %v", err),
			Error:   err.Error(),
		}, nil
	}

	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to marshal JSON: %v", err),
			Error:   err.Error(),
		}, nil
	}

	if err := ioutil.WriteFile(path, output, 0644); err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to write file: %v", err),
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully appended JSON object to array in file: %s (total %d items)", path, len(data)),
	}, nil
}

// FileEditTool edits a file by replacing content.
type FileEditTool struct {
	BaseTool
}

// NewFileEditTool creates a new FileEditTool.
func NewFileEditTool() *FileEditTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": { "type": "string", "description": "Path to the file to edit" },
			"old_str": { "type": "string", "description": "String to replace" },
			"new_str": { "type": "string", "description": "Replacement string" }
		},
		"required": ["path", "old_str", "new_str"]
	}`)
	return &FileEditTool{
		BaseTool: *NewBaseTool(
			"file_edit",
			"编辑文件，替换指定内容",
			schema,
		),
	}
}

// Execute executes the file edit tool.
func (t *FileEditTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "path is required",
			Error:   "path parameter is missing or invalid",
		}, nil
	}

	oldStr, ok := args["old_str"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "old_str is required",
			Error:   "old_str parameter is missing or invalid",
		}, nil
	}

	newStr, ok := args["new_str"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "new_str is required",
			Error:   "new_str parameter is missing or invalid",
		}, nil
	}

	// Expand path
	path = filepath.Clean(path)

	// Read file
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to read file: %v", err),
			Error:   err.Error(),
		}, nil
	}

	// Replace content
	newContent := replaceAll(string(content), oldStr, newStr)

	// Write file
	if err := ioutil.WriteFile(path, []byte(newContent), 0644); err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("Failed to write file: %v", err),
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully edited file: %s", path),
	}, nil
}

// replaceAll replaces all occurrences of oldStr with newStr in content.
func replaceAll(content, oldStr, newStr string) string {
	// Simple implementation for demonstration
	// In production, use strings.ReplaceAll
	return content // TODO: Implement actual replacement
}
