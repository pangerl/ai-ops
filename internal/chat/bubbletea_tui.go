package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-ops/internal/config"
	"ai-ops/internal/llm"
	"ai-ops/internal/tools"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Message 代表一条聊天消息
type Message struct {
	Content   string
	IsUser    bool
	Timestamp time.Time
	Thinking  string // AI的思考过程
}

// BubbleTeaModel 是新的聊天界面模型
type BubbleTeaModel struct {
	// 组件
	viewport viewport.Model
	textarea textarea.Model

	// 状态
	messages   []Message
	ready      bool
	quitting   bool
	processing bool
	width      int
	height     int

	// AI相关
	client      llm.ModelAdapter
	toolManager tools.ToolManager
	session     *Session

	// 渲染器
	renderer *glamour.TermRenderer

	// 样式
	userStyle     lipgloss.Style
	aiStyle       lipgloss.Style
	thinkingStyle lipgloss.Style
	systemStyle   lipgloss.Style
	inputStyle    lipgloss.Style
	helpStyle     lipgloss.Style
}

// chatProcessingMsg 表示正在处理AI响应
type chatProcessingMsg struct{}

// chatResponseMsg 包含AI的响应
type chatResponseMsg struct {
	response string
	err      error
}

// NewBubbleTeaModel 创建新的Bubble Tea聊天模型
func NewBubbleTeaModel(client llm.ModelAdapter, toolManager tools.ToolManager, config SessionConfig) (*BubbleTeaModel, error) {
	// 创建textarea
	ta := textarea.New()
	ta.Placeholder = "输入您的消息... (Ctrl+S 发送，Ctrl+C 退出)"
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetKeys("enter")

	// 创建viewport
	vp := viewport.New(80, 20)
	vp.SetContent("")

	// 创建渲染器
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return nil, fmt.Errorf("创建Markdown渲染器失败: %w", err)
	}

	// 定义样式
	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).
		Bold(true).
		MarginLeft(1)

	aiStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ffff")).
		Bold(true).
		MarginLeft(1)

	thinkingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffff00")).
		Italic(true).
		MarginLeft(2)

	systemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		MarginLeft(1)

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#04B575")).
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		MarginLeft(1)

	m := &BubbleTeaModel{
		viewport:      vp,
		textarea:      ta,
		messages:      make([]Message, 0),
		client:        client,
		toolManager:   toolManager,
		session:       NewSession(client, toolManager, config),
		renderer:      renderer,
		userStyle:     userStyle,
		aiStyle:       aiStyle,
		thinkingStyle: thinkingStyle,
		systemStyle:   systemStyle,
		inputStyle:    inputStyle,
		helpStyle:     helpStyle,
	}

	// 添加欢迎消息
	m.addWelcomeMessage()

	return m, nil
}

// addWelcomeMessage 添加欢迎消息
func (m *BubbleTeaModel) addWelcomeMessage() {
	mode := "普通对话模式"
	tips := "我是你的AI助手，随时为你答疑解惑"

	if m.session.config.Mode == "agent" {
		mode = "智能体模式"
		tips = "我是自主智能体，能够分析任务并制定执行计划"
	}

	welcomeMsg := fmt.Sprintf("🤖 欢迎来到AI对话系统 - %s\n%s", mode, tips)
	if m.session.config.ShowThinking {
		welcomeMsg += "\n💭 思考过程显示已开启"
	}
	welcomeMsg += "\n\n💡 快捷键提示：\n" +
		"  • Ctrl+S - 发送消息\n" +
		"  • Enter - 换行\n" +
		"  • Ctrl+C - 退出程序\n" +
		"  • Ctrl+L - 清空历史\n" +
		"  • Ctrl+U/Ctrl+D - 滚动消息历史"

	m.messages = append(m.messages, Message{
		Content:   welcomeMsg,
		IsUser:    false,
		Timestamp: time.Now(),
	})

	m.updateViewport()
}

// Init 初始化模型
func (m *BubbleTeaModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update 处理消息更新
func (m *BubbleTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width

		// 调整各组件大小
		headerHeight := 1
		helpHeight := 3
		inputHeight := 5
		viewportHeight := m.height - headerHeight - helpHeight - inputHeight - 2

		m.viewport.Width = m.width - 2
		m.viewport.Height = viewportHeight

		m.textarea.SetWidth(m.width - 4)
		m.textarea.SetHeight(3)

		if !m.ready {
			m.ready = true
		}

		m.updateViewport()

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit

		case msg.Type == tea.KeyCtrlL:
			// 清空历史
			m.messages = make([]Message, 0)
			m.addWelcomeMessage()
			return m, nil

		case msg.Type == tea.KeyCtrlS:
			// 发送消息
			return m.sendMessage()

		case msg.Type == tea.KeyCtrlU:
			// 向上滚动
			m.viewport.LineUp(5)
			return m, nil

		case msg.Type == tea.KeyCtrlD:
			// 向下滚动
			m.viewport.LineDown(5)
			return m, nil

			// 其他键盘事件由子组件处理
		}

	case chatProcessingMsg:
		// 显示处理状态
		return m, nil

	case chatResponseMsg:
		// 处理AI响应
		m.processing = false
		if msg.err != nil {
			m.addErrorMessage(fmt.Sprintf("错误: %v", msg.err))
		} else {
			m.addAIMessage(msg.response)
		}
		return m, nil
	}

	// 更新子组件
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

// View 渲染界面
func (m *BubbleTeaModel) View() string {
	if m.quitting {
		return "再见!\n"
	}

	if !m.ready {
		return "\n正在初始化..."
	}

	// 标题栏
	header := m.systemStyle.Render("AI-OPS 智能运维助手")

	// 消息区域
	messagesView := m.viewport.View()

	// 处理状态
	var statusLine string
	if m.processing {
		statusLine = m.systemStyle.Render("🤔 AI正在思考...")
	} else {
		statusLine = ""
	}

	// 输入区域
	inputArea := m.inputStyle.Render(m.textarea.View())

	// 帮助信息
	help := m.helpStyle.Render("Ctrl+S: 发送 | Ctrl+C: 退出 | Ctrl+L: 清空 | Ctrl+U/D: 滚动")

	// 组合所有部分
	var sections []string
	sections = append(sections, header)
	sections = append(sections, messagesView)
	if statusLine != "" {
		sections = append(sections, statusLine)
	}
	sections = append(sections, inputArea)
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}

// addUserMessage 添加用户消息
func (m *BubbleTeaModel) addUserMessage(content string) {
	m.messages = append(m.messages, Message{
		Content:   content,
		IsUser:    true,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}

// addAIMessage 添加AI消息
func (m *BubbleTeaModel) addAIMessage(content string) {
	var thinking string
	var actualContent string

	if m.session.config.ShowThinking {
		thinkingResult := ExtractThinking(content)
		thinking = thinkingResult.Thinking
		actualContent = thinkingResult.Content
	} else {
		actualContent = RemoveThinking(content)
	}

	m.messages = append(m.messages, Message{
		Content:   actualContent,
		IsUser:    false,
		Timestamp: time.Now(),
		Thinking:  thinking,
	})
	m.updateViewport()
}

// addErrorMessage 添加错误消息
func (m *BubbleTeaModel) addErrorMessage(content string) {
	m.messages = append(m.messages, Message{
		Content:   content,
		IsUser:    false,
		Timestamp: time.Now(),
	})
	m.updateViewport()
}

// updateViewport 更新视口内容
func (m *BubbleTeaModel) updateViewport() {
	var content strings.Builder

	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n")
		}

		// 时间戳
		timestamp := msg.Timestamp.Format("15:04:05")

		if msg.IsUser {
			// 用户消息 - 左对齐布局
			header := m.userStyle.Render(fmt.Sprintf("You [%s]:", timestamp))
			content.WriteString(header + "\n")

			// 用户消息气泡 - 左对齐
			maxWidth := min(m.width*4/5, 80)
			if maxWidth <= 0 {
				maxWidth = 80
			}

			bubbleStyle := lipgloss.NewStyle().
				Padding(0, 1).
				MarginLeft(1).
				Width(maxWidth).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00ff00"))

			userBubble := bubbleStyle.Render(msg.Content)
			content.WriteString(userBubble + "\n")
		} else {
			// AI消息
			if msg.Thinking != "" && m.session.config.ShowThinking {
				// 显示思考过程
				thinkingHeader := m.thinkingStyle.Render("🤔 思考过程:")
				content.WriteString(thinkingHeader + "\n")

				rendered, err := m.renderer.Render(msg.Thinking)
				if err != nil {
					content.WriteString(m.thinkingStyle.Render(msg.Thinking))
				} else {
					content.WriteString(rendered)
				}
				content.WriteString("\n---\n")
			}

			// AI回答
			header := m.aiStyle.Render(fmt.Sprintf("AI [%s]:", timestamp))
			content.WriteString(header + "\n")

			// AI消息气泡 - 左对齐
			maxWidth := min(m.width*4/5, 80)
			if maxWidth <= 0 {
				maxWidth = 80
			}

			aiBubbleStyle := lipgloss.NewStyle().
				Padding(0, 1).
				MarginLeft(1).
				Width(maxWidth).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#00ffff"))

			rendered, err := m.renderer.Render(msg.Content)
			var aiContentText string
			if err != nil {
				aiContentText = msg.Content
			} else {
				aiContentText = rendered
			}

			aiContent := aiBubbleStyle.Render(aiContentText)
			content.WriteString(aiContent + "\n")
		}

		content.WriteString("\n")
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

// sendMessage 发送消息的统一处理方法
func (m *BubbleTeaModel) sendMessage() (tea.Model, tea.Cmd) {
	if m.processing {
		return m, nil
	}

	input := strings.TrimSpace(m.textarea.Value())
	if input == "" {
		return m, nil
	}

	// 清空输入框
	m.textarea.Reset()

	// 添加用户消息
	m.addUserMessage(input)

	// 开始处理
	m.processing = true
	return m, tea.Batch(
		tea.Cmd(func() tea.Msg { return chatProcessingMsg{} }),
		m.processUserMessage(input),
	)
}

// processUserMessage 处理用户消息
func (m *BubbleTeaModel) processUserMessage(input string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		timeout := time.Duration(config.GetConfig().AI.Timeout) * time.Second
		if timeout <= 0 {
			timeout = 2 * time.Minute
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		response, err := m.session.ProcessMessage(ctx, input)
		return chatResponseMsg{response: response, err: err}
	})
}

// RunBubbleTeaChat 启动新的Bubble Tea聊天界面
func RunBubbleTeaChat(client llm.ModelAdapter, toolManager tools.ToolManager, config SessionConfig) error {
	model, err := NewBubbleTeaModel(client, toolManager, config)
	if err != nil {
		return fmt.Errorf("初始化聊天界面失败: %w", err)
	}

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

// RunChat 启动对话（入口函数）
func RunChat(client llm.ModelAdapter, toolManager tools.ToolManager, config SessionConfig) {
	// 直接使用 Bubble Tea 界面
	err := RunBubbleTeaChat(client, toolManager, config)
	if err != nil {
		fmt.Printf("启动界面失败: %v\n", err)
		return
	}
}
