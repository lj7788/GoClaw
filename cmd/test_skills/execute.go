package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	"github.com/zeroclaw-labs/goclaw/pkg/skills"
)

func main() {
	// 创建技能加载器
	skillsDir := "/Users/haha/.zeroclaw/workspace/skills"
	loader := skills.NewSkillLoader(skillsDir)

	// 加载所有技能
	if err := loader.LoadSkills(); err != nil {
		log.Fatalf("Failed to load skills: %v", err)
	}

	// 获取 stock-analyzer 技能
	skill, exists := loader.GetSkill("stock-analyzer")
	if !exists {
		log.Fatalf("Stock analyzer skill not found")
	}

	fmt.Printf("Testing stock-analyzer skill:\n")
	fmt.Printf("Name: %s\n", skill.Name)
	fmt.Printf("Description: %s\n", skill.Description)

	// 创建技能执行器
	executor := skills.NewSkillExecutor()

	// 注册 stock-analyzer 的处理器
	executor.RegisterHandler("stock-analyzer", func(ctx context.Context, skill *skills.Skill, command string, args map[string]interface{}) (string, error) {
		// 获取技能路径
		skillPath := filepath.Join(skillsDir, skill.Name+"-skill")

		// 构建命令参数
		scriptPath := filepath.Join(skillPath, "analyze.sh")
		cmdArgs := []string{}

		// 添加参数
		if stock, ok := args["stock"].(string); ok {
			cmdArgs = append(cmdArgs, "--stock", stock)
		}
		if market, ok := args["market"].(string); ok {
			cmdArgs = append(cmdArgs, "--market", market)
		}

		// 执行脚本
		cmd := exec.CommandContext(ctx, scriptPath, cmdArgs...)
		cmd.Dir = skillPath

		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("execution failed: %w\nOutput: %s", err, string(output))
		}

		return string(output), nil
	})

	// 测试执行 analyze 命令
	ctx := context.Background()
	args := map[string]interface{}{
		"stock":  "00700",
		"market": "hk",
	}

	fmt.Printf("\nExecuting analyze command...\n")
	result, err := executor.Execute(ctx, skill, "analyze", args)
	if err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}

	fmt.Printf("Result:\n")
	fmt.Printf("%s\n", result)
}