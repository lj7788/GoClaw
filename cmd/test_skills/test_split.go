package main

import (
	"fmt"
	"strings"
)

func main() {
	line := "- **Analyze Stock** (analyze): 分析指定股票，生成投资分析报告"
	
	fmt.Printf("Original line: %s\n", line)
	
	// Split by **
	parts := strings.Split(line, "**")
	fmt.Printf("Split by **: %d parts\n", len(parts))
	for i, part := range parts {
		fmt.Printf("  parts[%d]: %s\n", i, part)
	}
	
	if len(parts) >= 3 {
		// The command name is in parentheses in parts[2]
		// parts[2] is like " (analyze): description"
		cmdName := ""
		if cmdNameStart := strings.Index(parts[2], "("); cmdNameStart != -1 {
			fmt.Printf("Found '(' at position %d in parts[2]\n", cmdNameStart)
			remaining := parts[2][cmdNameStart:]
			fmt.Printf("Remaining after '(': %s\n", remaining)
			
			if cmdNameEnd := strings.Index(remaining, ")"); cmdNameEnd != -1 {
				cmdName = strings.TrimSpace(remaining[1:cmdNameEnd])
				fmt.Printf("Extracted command name: %s\n", cmdName)
			}
		}
		
		cmdDesc := ""
		if len(parts) >= 3 {
			cmdDesc = strings.TrimSpace(strings.TrimPrefix(parts[2], "):"))
			fmt.Printf("Extracted command description: %s\n", cmdDesc)
		}
		
		fmt.Printf("Final result:\n")
		fmt.Printf("  Command name: %s\n", cmdName)
		fmt.Printf("  Command description: %s\n", cmdDesc)
	}
}