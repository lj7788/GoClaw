// Package tools provides tool functionality for GoClaw.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zeroclaw-labs/goclaw/pkg/cron"
)

// CronJob represents a scheduled job.
type CronJob struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Expression  string                 `json:"expression"`
	Command     string                 `json:"command"`
	Type        string                 `json:"type"`
	NextRun     time.Time              `json:"next_run"`
	LastRun     *time.Time             `json:"last_run,omitempty"`
	LastStatus  string                 `json:"last_status,omitempty"`
	LastOutput  string                 `json:"last_output,omitempty"`
	Enabled     bool                   `json:"enabled"`
	OneShot     bool                   `json:"one_shot"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CronAddTool creates scheduled cron jobs.
type CronAddTool struct {
	BaseTool
	workspaceDir  string
	gatewayHost   string
	gatewayPort   int
}

// NewCronAddTool creates a new CronAddTool.
func NewCronAddTool(workspaceDir string, gatewayHost string, gatewayPort int) *CronAddTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"name": { "type": "string", "description": "任务名称，用于标识这个定时任务" },
			"expression": { "type": "string", "description": "Cron 表达式（例如：'0 16 * * 1-5' 表示工作日每天16:00）或 'at:YYYY-MM-DDTHH:MM' 格式用于一次性任务" },
			"type": { "type": "string", "description": "任务类型: shell(默认), python, nodejs, agent", "enum": ["shell", "python", "nodejs", "agent"] },
			"command": { "type": "string", "description": "要执行的 Shell 命令或脚本（与 agent_task 二选一）" },
			"agent_task": { "type": "string", "description": "要让 Agent 执行的任务描述（与 command 二选一）。例如：'分析股票爱尔眼科并发送到企业微信'" },
			"enabled": { "type": "boolean", "description": "是否启用任务（默认：true）" }
		},
		"required": ["expression"]
	}`)
	return &CronAddTool{
		BaseTool: *NewBaseTool(
			"cron_add",
			"创建定时任务。用于设置定期执行的任务。支持两种模式：1) 执行 Shell 命令（使用 command 参数）；2) 让 Agent 执行任务（使用 agent_task 参数）。支持标准 cron 表达式格式：分 时 日 月 周。例如：'0 16 * * 1-5' 表示工作日每天下午16:00执行，'*/10 * * * *' 表示每10分钟执行一次。",
			schema,
		),
		workspaceDir: workspaceDir,
		gatewayHost:  gatewayHost,
		gatewayPort:  gatewayPort,
	}
}

// Execute executes the cron add tool.
func (t *CronAddTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	expression, _ := args["expression"].(string)
	command, _ := args["command"].(string)
	agentTask, _ := args["agent_task"].(string)
	name, _ := args["name"].(string)
	taskType, _ := args["type"].(string)
	enabled := true
	if e, ok := args["enabled"].(bool); ok {
		enabled = e
	}

	if expression == "" {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "expression parameter is required",
		}, nil
	}
	
	if command == "" && agentTask == "" && taskType == "" {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "必须提供 command、agent_task 或 type 参数",
		}, nil
	}

	var actualCommand string
	var jobType string
	if agentTask != "" {
		jobType = "agent"
		actualCommand = agentTask
		if name == "" {
			name = agentTask
		}
	} else if taskType != "" {
		jobType = taskType
		actualCommand = command
	} else {
		jobType = "shell"
		actualCommand = command
	}

	scheduler := cron.GetScheduler(t.workspaceDir, t.gatewayHost, t.gatewayPort)
	if !scheduler.IsRunning() {
		if err := scheduler.Start(); err != nil {
			return &ToolResult{
				Success: false,
				Output:  "",
				Error:   fmt.Sprintf("Failed to start scheduler: %v", err),
			}, nil
		}
	}

	job := &cron.Job{
		Name:       name,
		Expression: expression,
		Command:    actualCommand,
		Type:       jobType,
		Enabled:    enabled,
		OneShot:    strings.HasPrefix(expression, "at:"),
	}

	if err := scheduler.AddJob(job); err != nil {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   fmt.Sprintf("Failed to add job: %v", err),
		}, nil
	}

	taskDesc := command
	if agentTask != "" {
		taskDesc = fmt.Sprintf("[Agent任务] %s", agentTask)
	}

	return &ToolResult{
		Success: true,
		Output: fmt.Sprintf(`已创建定时任务:
  ID: %s
  名称: %s
  类型: %s
  Cron表达式: %s
  任务内容: %s
  下次执行: %s
  状态: %v`,
			job.ID, job.Name, taskType, job.Expression, taskDesc, job.NextRun.Format("2006-01-02 15:04:05"), map[bool]string{true: "已启用", false: "已禁用"}[job.Enabled]),
	}, nil
}

func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (t *CronAddTool) parseNextRun(expr string) time.Time {
	if strings.HasPrefix(expr, "at:") {
		t, err := time.Parse(time.RFC3339, strings.TrimPrefix(expr, "at:"))
		if err == nil {
			return t
		}
	}
	return time.Now().Add(5 * time.Minute)
}

// CronListTool lists all scheduled cron jobs.
type CronListTool struct {
	BaseTool
	workspaceDir  string
	gatewayHost   string
	gatewayPort   int
}

// NewCronListTool creates a new CronListTool.
func NewCronListTool(workspaceDir string, gatewayHost string, gatewayPort int) *CronListTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {}
	}`)
	return &CronListTool{
		BaseTool: *NewBaseTool(
			"cron_list",
			"列出所有定时任务。返回所有已创建的定时任务列表，包括任务ID、名称、执行表达式、上次运行状态等信息。",
			schema,
		),
		workspaceDir: workspaceDir,
		gatewayHost:  gatewayHost,
		gatewayPort:  gatewayPort,
	}
}

// Execute executes the cron list tool.
func (t *CronListTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	scheduler := cron.GetScheduler(t.workspaceDir, t.gatewayHost, t.gatewayPort)
	jobs := scheduler.ListJobs()

	if len(jobs) == 0 {
		return &ToolResult{
			Success: true,
			Output:  "No scheduled jobs.",
		}, nil
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("Scheduled jobs (%d):", len(jobs)))
	for _, job := range jobs {
		status := "enabled"
		if !job.Enabled {
			status = "disabled"
		}
		oneShot := ""
		if job.OneShot {
			oneShot = " [one-shot]"
		}
		lastRun := "never"
		if job.LastRun != nil {
			lastRun = job.LastRun.Format(time.RFC3339)
		}
		lastStatus := "n/a"
		if job.LastStatus != "" {
			lastStatus = job.LastStatus
		}
		lines = append(lines, fmt.Sprintf("  - %s | %s | next: %s | last: %s (%s) [%s]%s",
			job.ID, job.Name, job.NextRun.Format(time.RFC3339), lastRun, lastStatus, status, oneShot))
		lines = append(lines, fmt.Sprintf("    Command: %s", job.Command))
	}

	return &ToolResult{
		Success: true,
		Output:  strings.Join(lines, "\n"),
	}, nil
}

// CronRemoveTool removes a scheduled job.
type CronRemoveTool struct {
	BaseTool
	workspaceDir  string
	gatewayHost   string
	gatewayPort   int
}

// NewCronRemoveTool creates a new CronRemoveTool.
func NewCronRemoveTool(workspaceDir string, gatewayHost string, gatewayPort int) *CronRemoveTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": { "type": "string", "description": "要删除的任务 ID" }
		},
		"required": ["id"]
	}`)
	return &CronRemoveTool{
		BaseTool: *NewBaseTool(
			"cron_remove",
			"删除指定的定时任务。需要提供任务 ID，可以使用 cron_list 工具查看所有任务及其 ID。",
			schema,
		),
		workspaceDir: workspaceDir,
		gatewayHost:  gatewayHost,
		gatewayPort:  gatewayPort,
	}
}

// Execute executes the cron remove tool.
func (t *CronRemoveTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "id parameter is required",
		}, nil
	}

	scheduler := cron.GetScheduler(t.workspaceDir, t.gatewayHost, t.gatewayPort)
	if err := scheduler.RemoveJob(id); err != nil {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   fmt.Sprintf("Job not found: %s", id),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Removed job: %s", id),
	}, nil
}

// CronRunTool runs a scheduled job immediately.
type CronRunTool struct {
	BaseTool
	workspaceDir  string
	gatewayHost   string
	gatewayPort   int
}

// NewCronRunTool creates a new CronRunTool.
func NewCronRunTool(workspaceDir string, gatewayHost string, gatewayPort int) *CronRunTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": { "type": "string", "description": "要查看的任务 ID" }
		},
		"required": ["id"]
	}`)
	return &CronRunTool{
		BaseTool: *NewBaseTool(
			"cron_run",
			"查看指定定时任务的详细信息，包括任务 ID、名称、执行命令、下次执行时间、上次执行时间和执行状态。",
			schema,
		),
		workspaceDir: workspaceDir,
		gatewayHost:  gatewayHost,
		gatewayPort:  gatewayPort,
	}
}

// Execute executes the cron run tool.
func (t *CronRunTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   "id parameter is required",
		}, nil
	}

	scheduler := cron.GetScheduler(t.workspaceDir, t.gatewayHost, t.gatewayPort)
	job, err := scheduler.GetJob(id)
	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  "",
			Error:   fmt.Sprintf("Job not found: %s", id),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output: fmt.Sprintf("Job %s is scheduled to run at %s\nCommand: %s\nLast run: %s\nLast status: %s",
			id, job.NextRun.Format(time.RFC3339), job.Command,
			func() string {
				if job.LastRun != nil {
					return job.LastRun.Format(time.RFC3339)
				}
				return "never"
			}(),
			func() string {
				if job.LastStatus != "" {
					return job.LastStatus
				}
				return "n/a"
			}()),
	}, nil
}
