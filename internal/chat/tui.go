package chat

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ai-ops/internal/config"
	ai "ai-ops/internal/llm"
	"ai-ops/internal/tools"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/glamour"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
)

// TUI represents the text user interface for the chat.
type TUI struct {
	client          ai.ModelAdapter
	toolManager     tools.ToolManager
	session         *Session
	rl              *readline.Instance
	userColor       *color.Color
	aiColor         *color.Color
	aiResponseColor *color.Color
	errorColor      *color.Color
	thinkingColor   *color.Color
	renderer        *glamour.TermRenderer
	thinkingRenderer *glamour.TermRenderer
}

// NewTUI creates a new TUI.
func NewTUI(client ai.ModelAdapter, toolManager tools.ToolManager, config SessionConfig) (*TUI, error) {
	userColor := color.New(color.FgGreen).Add(color.Bold)
	aiColor := color.New(color.FgCyan)
	aiResponseColor := color.New(color.FgHiWhite)
	errorColor := color.New(color.FgRed)
	thinkingColor := color.New(color.FgYellow).Add(color.Italic)

	historyFile := "/tmp/ai-ops-readline.tmp"
	homeDir, err := os.UserHomeDir()
	if err == nil {
		historyFile = filepath.Join(homeDir, ".ai-ops-history")
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          userColor.Sprint("You: "),
		HistoryFile:     historyFile,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, fmt.Errorf("创建读取行实例失败: %w", err)
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return nil, fmt.Errorf("创建Markdown渲染器失败: %w", err)
	}

	// 为思考内容创建单独的渲染器（较暗的样式）
	thinkingRenderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return nil, fmt.Errorf("创建思考渲染器失败: %w", err)
	}

	tui := &TUI{
		client:           client,
		toolManager:      toolManager,
		session:          NewSession(client, toolManager, config),
		rl:               rl,
		userColor:        userColor,
		aiColor:          aiColor,
		aiResponseColor:  aiResponseColor,
		errorColor:       errorColor,
		thinkingColor:    thinkingColor,
		renderer:         renderer,
		thinkingRenderer: thinkingRenderer,
	}

	return tui, nil
}

// Run starts the main chat loop.
func (t *TUI) Run() {
	defer t.rl.Close()
	t.showWelcome()

	// 显示当前配置
	t.showConfiguration()

	for {
		userInput, err := t.rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			if err == io.EOF || err == readline.ErrInterrupt {
				break
			}
			t.errorColor.Printf("读取输入失败: %v\n", err)
			continue
		}

		if err := t.processInput(userInput); err != nil {
			if err == io.EOF {
				break
			}
			t.errorColor.Printf("处理消息失败: %v\n", err)
		}
	}
	fmt.Println("再见!")
}

// processInput handles the user's input.
func (t *TUI) processInput(input string) error {
	trimmedInput := strings.TrimSpace(input)
	switch trimmedInput {
	case "exit", "quit", "bye":
		return io.EOF
	case "help":
		t.printHelp()
		return nil
	case "":
		return nil
	}

	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " AI 正在思考..."
	s.Start()
	defer s.Stop()

	timeout := time.Duration(config.GetConfig().AI.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 2 * time.Minute // 如果未设置或无效，则回退到默认值
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	finalResponse, err := t.session.ProcessMessage(ctx, input)
	if err != nil {
		return err
	}

	// 处理思考过程和响应内容
	t.renderResponseWithThinking(finalResponse)
	fmt.Println("---------------------------------------------------")
	return nil
}

// printHelp displays the help message.
func (t *TUI) printHelp() {
	mode := "普通对话"
	if t.session.config.Mode == "agent" {
		mode = "智能体"
	}

	thinkStatus := "关闭"
	if t.session.config.ShowThinking {
		thinkStatus = "开启"
	}

	fmt.Printf("\n当前配置:\n")
	fmt.Printf("  模式: %s\n", mode)
	fmt.Printf("  思考显示: %s\n", thinkStatus)
	fmt.Printf("\n可用命令:\n")
	fmt.Printf("  exit, quit, bye    - 退出程序\n")
	fmt.Printf("  help               - 显示此帮助信息\n")
	fmt.Printf("  clear              - 清空对话历史\n")
	fmt.Printf("---------------------------------------------------\n")
}

// showWelcome 显示欢迎信息
func (t *TUI) showWelcome() {
	mode := "普通对话模式"
	tips := "我是你的AI助手，随时为你答疑解惑"

	if t.session.config.Mode == "agent" {
		mode = "智能体模式"
		tips = "我是自主智能体，能够分析任务并制定执行计划"
	}

	fmt.Printf("欢迎来到AI对话系统 - %s\n", mode)
	fmt.Printf("%s\n", tips)
	if t.session.config.ShowThinking {
		fmt.Printf("💭 思考过程显示已开启\n")
	}
	fmt.Printf("输入 'help' 获取帮助，'exit' 退出\n")
}

// showConfiguration 显示当前配置
func (t *TUI) showConfiguration() {
	fmt.Printf("---------------------------------------------------\n")
}

// renderResponseWithThinking 渲染包含思考过程的响应
func (t *TUI) renderResponseWithThinking(response string) {
	if t.session.config.ShowThinking {
		thinking := ExtractThinking(response)
		if thinking.Thinking != "" {
			// 显示思考过程
			t.thinkingColor.Println("\n🤔 思考过程:")
			thinkingRendered, err := t.thinkingRenderer.Render(thinking.Thinking)
			if err != nil {
				fmt.Println(t.thinkingColor.Sprint(thinking.Thinking))
			} else {
				fmt.Print(thinkingRendered)
			}
			fmt.Println("---")
		}
		// 显示正式回答
		t.aiColor.Println("AI:")
		t.renderContent(thinking.Content)
	} else {
		// 移除思考标记，只显示正式内容
		content := RemoveThinking(response)
		t.aiColor.Println("\nAI:")
		t.renderContent(content)
	}
}

// renderContent 渲染内容
func (t *TUI) renderContent(content string) {
	renderedOutput, err := t.renderer.Render(content)
	if err != nil {
		t.errorColor.Printf("渲染Markdown失败: %v\n", err)
		// Fallback to plain text
		fmt.Println(t.aiResponseColor.Sprint(content))
	} else {
		fmt.Print(renderedOutput)
	}
}

// RunChat 启动对话（新的入口函数）
func RunChat(client ai.ModelAdapter, toolManager tools.ToolManager, config SessionConfig) {
	tui, err := NewTUI(client, toolManager, config)
	if err != nil {
		fmt.Printf("初始化TUI失败: %v\n", err)
		return
	}
	tui.Run()
}

// RunSimpleLoop 保持向后兼容（废弃）
func RunSimpleLoop(client ai.ModelAdapter, toolManager tools.ToolManager) {
	config := SessionConfig{
		Mode:         "chat",
		ShowThinking: false,
	}
	RunChat(client, toolManager, config)
}
