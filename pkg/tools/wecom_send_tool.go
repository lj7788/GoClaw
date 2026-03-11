package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type WecomSendFunc func(ctx context.Context, chatID, content string) error

type WecomSendTool struct {
	BaseTool
	sendFunc   WecomSendFunc
	defaultTo  string
}

func NewWecomSendTool(sendFunc WecomSendFunc, defaultTo string) *WecomSendTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"message": {
				"type": "string",
				"description": "要发送的消息内容（支持Markdown格式）"
			},
			"chat_id": {
				"type": "string",
				"description": "接收消息的会话ID（用户ID或群ID，可选，不填则使用默认配置）"
			}
		},
		"required": ["message"]
	}`)
	return &WecomSendTool{
		BaseTool: *NewBaseTool(
			"wecom_send",
			"发送消息到企业微信。通过WebSocket长连接主动发送消息给指定用户或群聊。",
			schema,
		),
		sendFunc:  sendFunc,
		defaultTo: defaultTo,
	}
}

func (t *WecomSendTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	message, _ := args["message"].(string)
	chatID, _ := args["chat_id"].(string)

	log.Printf("[WecomSendTool] Execute called: message=%s, chatID=%s, defaultTo=%s", message, chatID, t.defaultTo)

	if message == "" {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "message 参数是必需的",
		}, nil
	}

	if t.sendFunc == nil {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "企业微信未配置或未连接。请确保已配置企业微信机器人并已启动。",
		}, nil
	}

	if chatID == "" {
		chatID = t.defaultTo
	}

	if chatID == "" {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "未指定接收者。请在参数中提供 chat_id 或在配置中设置 default_to。",
		}, nil
	}

	log.Printf("[WecomSendTool] Calling sendFunc with chatID=%s", chatID)
	err := t.sendFunc(ctx, chatID, message)
	if err != nil {
		log.Printf("[WecomSendTool] sendFunc error: %v", err)
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   fmt.Sprintf("发送消息失败: %v。请确保使用 daemon 命令启动以连接企业微信。", err),
		}, nil
	}

	log.Printf("[WecomSendTool] Message sent successfully")
	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("消息已成功发送到企业微信会话: %s", chatID),
	}, nil
}
