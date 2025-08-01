package chat

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"ai-ops/internal/ai"
	"ai-ops/internal/tools"

	"github.com/briandowns/spinner"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// RunSimpleLoop 启动一个简单的交互式对话循环
func RunSimpleLoop(client ai.AIClient, toolManager tools.ToolManager) {
	userColor := color.New(color.FgGreen).Add(color.Bold)
	aiColor := color.New(color.FgCyan)
	aiResponseColor := color.New(color.FgHiWhite)
	errorColor := color.New(color.FgRed)

	fmt.Println("欢迎来到AI对话模式。输入 'exit' 或 'quit' 退出。")
	fmt.Println("---------------------------------------------------")

	session := NewSession(client, toolManager)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          userColor.Sprint("You: "),
		HistoryFile:     "/tmp/ai-ops-readline.tmp",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		userInput, err := rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			if err == io.EOF || err == readline.ErrInterrupt {
				break
			}
			errorColor.Printf("读取输入失败: %v\n", err)
			continue
		}

		if userInput == "exit" || userInput == "quit" {
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

		finalResponse, err := session.ProcessMessage(ctx, userInput)
		s.Stop()

		if err != nil {
			errorColor.Printf("处理消息失败: %v\n", err)
			continue
		}

		aiColor.Println("AI:")
		fmt.Println(aiResponseColor.Sprint(finalResponse))
		fmt.Println("---------------------------------------------------")
	}

	fmt.Println("再见!")
}
