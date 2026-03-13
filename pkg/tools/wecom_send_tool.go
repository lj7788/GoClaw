package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

	recipients := strings.Split(chatID, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	var failed []string
	var success []string

	for _, recipient := range recipients {
		if recipient == "" {
			continue
		}

		log.Printf("[WecomSendTool] Sending to recipient: %s", recipient)
		err := t.sendFunc(ctx, recipient, message)
		if err != nil {
			log.Printf("[WecomSendTool] sendFunc FAILED for %s: %v", recipient, err)
			failed = append(failed, recipient)
		} else {
			log.Printf("[WecomSendTool] sendFunc SUCCESS for %s", recipient)
			success = append(success, recipient)
		}
	}

	if len(failed) > 0 && len(success) == 0 {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   fmt.Sprintf("发送消息失败。失败接收者: %s", strings.Join(failed, ", ")),
		}, nil
	}

	resultMsg := fmt.Sprintf("消息已成功发送到: %s", strings.Join(success, ", "))
	if len(failed) > 0 {
		resultMsg += fmt.Sprintf("\n失败接收者: %s", strings.Join(failed, ", "))
	}

	log.Printf("[WecomSendTool] Message sent: %s", resultMsg)
	return &ToolResult{
		Success: true,
		Output:  resultMsg,
	}, nil
}
