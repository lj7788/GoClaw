package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
)

// StockAnalyzerTool analyzes stocks using the stock-analyzer skill.
type StockAnalyzerTool struct {
	BaseTool
	skillsDir string
}

// NewStockAnalyzerTool creates a new StockAnalyzerTool.
func NewStockAnalyzerTool(skillsDir string) *StockAnalyzerTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"stock": {
				"type": "string",
				"description": "股票名称或代码（如：贵州茅台、600519、00700、AAPL、MU）"
			},
			"market": {
				"type": "string",
				"description": "市场类型（可选：sh-沪市、sz-深市、hk-港股、us-美股，默认自动识别）",
				"enum": ["sh", "sz", "hk", "us", "auto"]
			}
		},
		"required": ["stock"]
	}`)
	return &StockAnalyzerTool{
		BaseTool: *NewBaseTool(
			"stock_analyze",
			"分析指定股票，生成投资分析报告。支持A股、港股、美股等东方财富覆盖的所有市场",
			schema,
		),
		skillsDir: skillsDir,
	}
}

// Execute executes the stock analyzer tool.
func (t *StockAnalyzerTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	stock, ok := args["stock"].(string)
	if !ok {
		return &ToolResult{
			Success: false,
			Output:  "stock is required",
			Error:   "stock parameter is missing or invalid",
		}, nil
	}

	market := "auto"
	if m, ok := args["market"].(string); ok {
		market = m
	}

	skillsDir := t.skillsDir
	if skillsDir == "" {
		skillsDir = "~/.zeroclaw/workspace/skills"
	}

	skillsDir = filepath.Clean(skillsDir)
	stockSkillDir := filepath.Join(skillsDir, "stock-analyzer-skill")

	cmd := exec.CommandContext(ctx, "node", "index.js", "--stock", stock, "--market", market)
	cmd.Dir = stockSkillDir

	output, err := cmd.CombinedOutput()

	if err != nil {
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("股票分析失败：%s\n输出：%s", err.Error(), string(output)),
			Error:   err.Error(),
		}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return &ToolResult{
			Success: true,
			Output:  string(output),
		}, nil
	}

	success, _ := result["success"].(bool)
	if !success {
		errorMsg, _ := result["error"].(string)
		return &ToolResult{
			Success: false,
			Output:  fmt.Sprintf("股票分析失败：%s", errorMsg),
			Error:   errorMsg,
		}, nil
	}

	report, _ := result["report"].(string)
	return &ToolResult{
		Success: true,
		Output:  report,
	}, nil
}
