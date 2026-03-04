package main

import (
	"fmt"
	"log"

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

	// 列出所有技能
	allSkills := loader.ListSkills()
	fmt.Printf("Found %d skills:\n", len(allSkills))
	for _, skill := range allSkills {
		fmt.Printf("- %s: %s\n", skill.Name, skill.Description)
	}

	// 测试获取特定技能
	if skill, exists := loader.GetSkill("stock-analyzer-skill"); exists {
		fmt.Printf("\nStock Analyzer Skill Details:\n")
		fmt.Printf("Name: %s\n", skill.Name)
		fmt.Printf("Description: %s\n", skill.Description)
		fmt.Printf("Version: %s\n", skill.Version)
		fmt.Printf("Commands: %d\n", len(skill.Commands))
		for _, cmd := range skill.Commands {
			fmt.Printf("  - %s: %s\n", cmd.Name, cmd.Description)
			if len(cmd.Parameters) > 0 {
				fmt.Printf("    Parameters:\n")
				for _, param := range cmd.Parameters {
					required := ""
					if param.Required {
						required = " (required)"
					}
					fmt.Printf("      - %s: %s%s\n", param.Name, param.Description, required)
				}
			}
		}
		fmt.Printf("Tools: %d\n", len(skill.Tools))
		for _, tool := range skill.Tools {
			fmt.Printf("  - %s: %s (kind: %s)\n", tool.Name, tool.Description, tool.Kind)
		}
	}
}