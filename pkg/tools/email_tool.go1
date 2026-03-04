// Package tools provides tool functionality for GoClaw.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// EmailTool sends emails using the email-sender skill.
type EmailTool struct {
	BaseTool
	skillsDir string
}

// NewEmailTool creates a new EmailTool.
func NewEmailTool(skillsDir string) *EmailTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"recipient": {
				"type": "string",
				"description": "收件人邮箱地址"
			},
			"subject": {
				"type": "string",
				"description": "邮件主题"
			},
			"body": {
				"type": "string",
				"description": "邮件正文内容"
			}
		},
		"required": ["recipient", "subject", "body"]
	}`)
	return &EmailTool{
		BaseTool: *NewBaseTool(
			"email_send",
			"发送邮件到指定收件人",
			schema,
		),
		skillsDir: skillsDir,
	}
}

// Execute executes the email tool.
func (t *EmailTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	recipient, ok := args["recipient"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "recipient is required",
			Error:   "recipient parameter is missing or invalid",
		}, nil
	}

	subject, ok := args["subject"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "subject is required",
			Error:   "subject parameter is missing or invalid",
		}, nil
	}

	body, ok := args["body"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "body is required",
			Error:   "body parameter is missing or invalid",
		}, nil
	}

	// Run the email-sender skill
	skillsDir := t.skillsDir
	if skillsDir == "" {
		skillsDir = "~/.zeroclaw/workspace/skills"
	}

	skillsDir = filepath.Clean(skillsDir)
	emailSkillDir := filepath.Join(skillsDir, "email-sender-skill")

	cmd := exec.CommandContext(ctx, "node", "index.js", "--to", recipient, "--subject", subject, "--body", body)
	cmd.Dir = emailSkillDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("邮件发送失败：%s\n输出：%s", err.Error(), string(output)),
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("邮件已成功发送到 %s\n主题：%s\n内容：%s", recipient, subject, body),
	}, nil
}