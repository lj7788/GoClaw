package main

import (
	"fmt"
	"strings"

	"github.com/zeroclaw-labs/goclaw/pkg/skills"
)

func main() {
	// 读取 SKILL.md 内容
	content := `# Stock Analyzer

## Description
全球股票综合分析工具。支持A股、港股、美股等东方财富覆盖的所有市场。根据用户输入的股票名称或代码，从东方财富网获取股票信息，进行基本面、新闻面、资金面三维分析，给出投资建议、买入价位和卖出价位。

## Commands
- **Analyze Stock** (analyze): 分析指定股票，生成投资分析报告
  - Parameters:
    - stock (required): 股票名称或代码（如：贵州茅台、600519、00700、AAPL、MU）
    - market (optional): 市场类型（sh-沪市、sz-深市、hk-港股、us-美股，默认自动识别）

## Features
- 支持A股、港股、美股等多市场分析
- 基本面、新闻面、资金面三维分析
- 生成Markdown和HTML两种格式的报告
- 提供投资建议、买入价位和卖出价位`

	// 解析命令
	var commands []skills.SkillCommand
	var tools []skills.SkillTool

	lines := strings.Split(content, "\n")
	inCommandsSection := false
	var currentCommand *skills.SkillCommand

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check if we're in the Commands section
		if strings.HasPrefix(trimmed, "## Commands") || strings.HasPrefix(trimmed, "### Commands") {
			inCommandsSection = true
			fmt.Println("Found Commands section")
			continue
		}
		
		// Exit commands section when we hit another major section
		if inCommandsSection && strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "## Commands") {
			inCommandsSection = false
			fmt.Println("Exiting Commands section")
			continue
		}

		// Parse command definitions
		if inCommandsSection {
			fmt.Printf("Processing line: %s\n", trimmed)
			// Look for command definition: **Command Name** (`command_name`): description
			if strings.HasPrefix(trimmed, "- **") && strings.Contains(trimmed, "** (") {
				// Extract command name and description
				parts := strings.Split(trimmed, "**")
				fmt.Printf("  Split into %d parts\n", len(parts))
				if len(parts) >= 3 {
					// Extract command name from parentheses
					cmdName := ""
					if cmdNameStart := strings.Index(parts[1], "("); cmdNameStart != -1 {
						if cmdNameEnd := strings.Index(parts[1][cmdNameStart:], ")"); cmdNameEnd != -1 {
							cmdName = strings.TrimSpace(parts[1][cmdNameStart+1 : cmdNameStart+cmdNameEnd])
							fmt.Printf("  Extracted command name: %s\n", cmdName)
						}
					}
					
					cmdDesc := ""
					if len(parts) >= 3 {
						cmdDesc = strings.TrimSpace(strings.TrimPrefix(parts[2], ":"))
						fmt.Printf("  Extracted command description: %s\n", cmdDesc)
					}
					
					if cmdName != "" {
						currentCommand = &skills.SkillCommand{
							Name:        cmdName,
							Description: cmdDesc,
						}
						commands = append(commands, *currentCommand)
						fmt.Printf("  Added command: %s\n", cmdName)
					}
				}
			} else if currentCommand != nil && strings.HasPrefix(trimmed, "  - ") {
				// Parse parameters
				paramLine := strings.TrimSpace(strings.TrimPrefix(trimmed, "  - "))
				if strings.Contains(paramLine, ":") {
					paramParts := strings.SplitN(paramLine, ":", 2)
					if len(paramParts) == 2 {
						paramName := strings.TrimSpace(paramParts[0])
						paramDesc := strings.TrimSpace(paramParts[1])
						currentCommand.Parameters = append(currentCommand.Parameters, skills.SkillParameter{
							Name:        paramName,
							Type:        "string",
							Description: paramDesc,
							Required:    strings.Contains(paramDesc, "(required)") || strings.Contains(paramDesc, "（必需）"),
						})
						fmt.Printf("  Added parameter: %s\n", paramName)
					}
				}
			}
		}
	}

	// Create tools from commands
	fmt.Printf("\nCreating tools from %d commands\n", len(commands))
	for _, cmd := range commands {
		tool := skills.SkillTool{
			Name:        cmd.Name,
			Description: cmd.Description,
			Kind:        "shell",
			Command:      fmt.Sprintf("./%s.sh", cmd.Name),
		}
		tools = append(tools, tool)
		fmt.Printf("  Created tool: %s\n", tool.Name)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("Commands: %d\n", len(commands))
	fmt.Printf("Tools: %d\n", len(tools))
}