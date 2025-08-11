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
	client          ai.AIClient
	toolManager     tools.ToolManager
	session         *Session
	rl              *readline.Instance
	userColor       *color.Color
	aiColor         *color.Color
	aiResponseColor *color.Color
	errorColor      *color.Color
	renderer        *glamour.TermRenderer
}

// NewTUI creates a new TUI.
func NewTUI(client ai.AIClient, toolManager tools.ToolManager) (*TUI, error) {
	userColor := color.New(color.FgGreen).Add(color.Bold)
	aiColor := color.New(color.FgCyan)
	aiResponseColor := color.New(color.FgHiWhite)
	errorColor := color.New(color.FgRed)

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
		return nil, fmt.Errorf("failed to create readline instance: %w", err)
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	return &TUI{
		client:          client,
		toolManager:     toolManager,
		session:         NewSession(client, toolManager),
		rl:              rl,
		userColor:       userColor,
		aiColor:         aiColor,
		aiResponseColor: aiResponseColor,
		errorColor:      errorColor,
		renderer:        renderer,
	}, nil
}

// Run starts the main chat loop.
func (t *TUI) Run() {
	defer t.rl.Close()
	fmt.Println("欢迎来到AI对话模式。输入 'exit', 'quit' 或 'bye' 退出。输入 'help' 获取帮助。")
	fmt.Println("---------------------------------------------------")

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

	t.aiColor.Println("\nAI:")
	renderedOutput, err := t.renderer.Render(finalResponse)
	if err != nil {
		t.errorColor.Printf("渲染Markdown失败: %v\n", err)
		// Fallback to plain text
		fmt.Println(t.aiResponseColor.Sprint(finalResponse))
	} else {
		fmt.Println(renderedOutput)
	}
	fmt.Println("---------------------------------------------------")
	return nil
}

// printHelp displays the help message.
func (t *TUI) printHelp() {
	fmt.Println("\n可用命令:")
	fmt.Println("  exit, quit, bye    - 退出程序")
	fmt.Println("  help               - 显示此帮助信息")
	fmt.Println("---------------------------------------------------")
}

// RunSimpleLoop initializes and runs the TUI.
func RunSimpleLoop(client ai.AIClient, toolManager tools.ToolManager) {
	tui, err := NewTUI(client, toolManager)
	if err != nil {
		fmt.Printf("初始化TUI失败: %v\n", err)
		return
	}
	tui.Run()
}
