package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ai-ops/internal/ai"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// RunSimpleLoop 启动一个简单的交互式对话循环
func RunSimpleLoop(client ai.AIClient) {
	scanner := bufio.NewScanner(os.Stdin)
	userColor := color.New(color.FgGreen).Add(color.Bold)
	aiColor := color.New(color.FgCyan)
	aiResponseColor := color.New(color.FgHiWhite)
	toolColor := color.New(color.FgYellow)
	errorColor := color.New(color.FgRed)

	fmt.Println("欢迎来到AI对话模式。输入 'exit' 或 'quit' 退出。")
	fmt.Println("---------------------------------------------------")

	for {
		userColor.Print("You: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				errorColor.Printf("读取输入失败: %v\n", err)
			}
			break
		}
		userInput := scanner.Text()

		if userInput == "exit" || userInput == "quit" {
			fmt.Println("再见!")
			break
		}

		if strings.TrimSpace(userInput) == "" {
			continue
		}

		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		s.Suffix = " AI 正在思考..."
		s.Start()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		response, err := client.SendMessage(ctx, userInput, nil)
		if err != nil {
			s.Stop()
			errorColor.Printf("调用AI失败: %v\n", err)
			continue
		}

		s.Stop()
		aiColor.Println("AI:")
		fmt.Println(aiResponseColor.Sprint(response.Content))

		if len(response.ToolCalls) > 0 {
			toolColor.Println("工具调用:")
			for _, tc := range response.ToolCalls {
				toolColor.Printf("  - 名称: %s\n", tc.Name)
				toolColor.Printf("    参数: %v\n", tc.Arguments)
			}
		}
		// say bye
		fmt.Println("---------------------------------------------------")

	}
}
