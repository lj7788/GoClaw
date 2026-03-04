package main

import (
	"encoding/json"
	"fmt"

	"github.com/zeroclaw-labs/goclaw/pkg/skills"
)

func main() {
	// 创建技能加载器
	skillsDir := "/Users/haha/.zeroclaw/workspace/skills"
	loader := skills.NewSkillLoader(skillsDir)

	// 加载所有技能
	if err := loader.LoadSkills(); err != nil {
		fmt.Printf("Failed to load skills: %v\n", err)
		return
	}

	// 获取 stock-analyzer-skill 技能
	if skill, exists := loader.GetSkill("stock-analyzer-skill"); exists {
		fmt.Printf("Skill: %s\n", skill.Name)
		fmt.Printf("Tools: %d\n", len(skill.Tools))
		
		for _, tool := range skill.Tools {
			fmt.Printf("\nTool: %s\n", tool.Name)
			fmt.Printf("Description: %s\n", tool.Description)
			fmt.Printf("Kind: %s\n", tool.Kind)
			fmt.Printf("Command: %s\n", tool.Command)
			fmt.Printf("Parameters: %d\n", len(tool.Parameters))
			for _, param := range tool.Parameters {
				fmt.Printf("  - %s: %s (required: %v)\n", param.Name, param.Description, param.Required)
			}
			
			// 创建执行器并获取参数模式
			executor := skills.NewSkillToolExecutor(skill, tool, skillsDir)
			spec := executor.Spec()
			fmt.Printf("\nSpec:\n")
			fmt.Printf("Name: %s\n", spec.Name)
			fmt.Printf("Description: %s\n", spec.Description)
			
			// 打印参数模式
			var schema map[string]interface{}
			if err := json.Unmarshal(spec.Parameters, &schema); err == nil {
				fmt.Printf("Parameters Schema:\n")
				schemaBytes, _ := json.MarshalIndent(schema, "", "  ")
				fmt.Printf("%s\n", string(schemaBytes))
			}
		}
	}
}