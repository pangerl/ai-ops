package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ai-ops/internal/llm"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// SessionConfig 会话配置
type SessionConfig struct {
	Mode         string // "chat" 或 "agent"
	ShowThinking bool   // 是否显示思考过程
}

// Session 管理一个独立的对话会话
type Session struct {
	client      llm.ModelAdapter
	toolManager tools.ToolManager
	messages    []llm.Message
	toolDefs    []tools.ToolDefinition
	config      SessionConfig
	maxHistory  int // 最大历史记录条数
}

// NewSession 创建一个新的对话会话
func NewSession(client llm.ModelAdapter, toolManager tools.ToolManager, config SessionConfig) *Session {
	session := &Session{
		client:      client,
		toolManager: toolManager,
		messages:    make([]llm.Message, 0),
		toolDefs:    toolManager.GetToolDefinitions(),
		config:      config,
		maxHistory:  10, // 默认保留最近10条消息
	}

	// 根据模式设置系统提示词
	systemPrompt := session.getSystemPrompt()
	if systemPrompt != "" {
		session.messages = append(session.messages, llm.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	return session
}

// ProcessMessage 处理用户输入并返回最终的 AI 响应
func (s *Session) ProcessMessage(ctx context.Context, userInput string) (string, error) {
	// 标记本轮对话的起始位置
	roundStartIndex := len(s.messages)
	// 将用户输入添加到消息历史
	s.messages = append(s.messages, llm.Message{Role: "user", Content: userInput})

	for {
		s.trimHistory()
		// 发送消息到 AI
		resp, err := s.client.SendMessage(ctx, s.messages, s.toolDefs)
		if err != nil {
			// 如果出错，从历史中移除最后一条消息，以备重试
			s.messages = s.messages[:len(s.messages)-1]
			return "", fmt.Errorf("发送消息到AI失败: %w", err)
		}

		// 调试：打印完整的 AI 响应
		respBytes, _ := json.Marshal(resp)
		util.Debugw("收到 AI 响应", map[string]any{"response": string(respBytes)})

		// 将 AI 的响应（不含工具调用）添加到历史记录
		aiResponseMsg := llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		s.messages = append(s.messages, aiResponseMsg)

		// 检查是否有工具调用需要执行
		if len(resp.ToolCalls) > 0 {
			// 需要调用工具
			toolResults, err := s.executeTools(ctx, resp.ToolCalls)
			if err != nil {
				return "", fmt.Errorf("执行工具失败: %w", err)
			}
			// 将工具结果添加到历史记录中，然后继续循环
			s.messages = append(s.messages, toolResults...)
			continue // 继续循环以获取最终的 AI 响应
		}

		// 如果没有工具调用，则根据 finish_reason 决定下一步操作
		switch resp.FinishReason {
		case "stop":
			// 对话完成，整合历史记录并返回最终内容
			s.consolidateHistory(roundStartIndex)
			return resp.Content, nil
		case "tool_calls":
			// 这种情况不应该发生，因为我们已经处理了工具调用
			// 但为了健壮性，我们返回一个错误
			return "", fmt.Errorf("unexpected state: finish_reason is 'tool_calls' but no tool calls were found")
		default:
			// 对于 Gemini，"STOP" 是一个有效的完成原因，即使没有工具调用
			if s.client.GetModelInfo().Type == "gemini" && resp.FinishReason == "STOP" {
				s.consolidateHistory(roundStartIndex)
				return resp.Content, nil
			}
			// 其他未知的 finish_reason
			return "", fmt.Errorf("unexpected finish_reason: %s", resp.FinishReason)
		}
	}
}

// trimHistory 修剪历史记录，以防止其无限增长
func (s *Session) trimHistory() {
	if len(s.messages) <= s.maxHistory {
		return
	}

	// 保留第一条消息（通常是初始提示或重要上下文）和最近的 maxHistory-1 条消息
	firstMessage := s.messages[0]
	recentMessages := s.messages[len(s.messages)-(s.maxHistory-1):]

	s.messages = make([]llm.Message, 0, s.maxHistory)
	s.messages = append(s.messages, firstMessage)
	s.messages = append(s.messages, recentMessages...)
}

// consolidateHistory 整合一轮对话的历史记录。
// 当一轮涉及工具调用的对话结束时，将中间步骤（工具调用、工具结果）替换为最终的用户问题和AI回答。
func (s *Session) consolidateHistory(roundStartIndex int) {
	// 如果本轮对话没有复杂的中间步骤（例如，只是 用户 -> AI），则无需整合
	// 一个需要整合的典型场景是：用户 -> AI(工具调用) -> 工具结果 -> AI(最终回答)
	// 所以消息数至少是4条，而本轮消息数是 len(s.messages) - roundStartIndex
	if len(s.messages)-roundStartIndex < 3 {
		return
	}

	// 获取本轮对话之前的历史记录
	previousHistory := s.messages[:roundStartIndex]
	// 获取本轮对话的初始用户消息
	userMessage := s.messages[roundStartIndex]
	// 获取本轮对话的最终AI回答
	finalAssistantMessage := s.messages[len(s.messages)-1]
	// 确保最终回答中不包含工具调用信息，因为它已经是最终文本
	finalAssistantMessage.ToolCalls = nil

	// 构建新的、整合后的历史记录
	newMessages := make([]llm.Message, 0, len(previousHistory)+2)
	newMessages = append(newMessages, previousHistory...)
	newMessages = append(newMessages, userMessage)
	newMessages = append(newMessages, finalAssistantMessage)

	s.messages = newMessages
	util.Debugw("历史记录已整合", map[string]any{"history_size": len(s.messages)})
}

// executeTools 执行工具调用并返回结果消息
func (s *Session) executeTools(ctx context.Context, toolCalls []llm.ToolCall) ([]llm.Message, error) {
	var toolMessages []llm.Message

	for _, tc := range toolCalls {
		result, err := s.toolManager.ExecuteToolCall(ctx, tools.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		})

		var content string
		if err != nil {
			// 将错误信息作为工具的返回结果
			content = fmt.Sprintf("Error executing tool %s: %v", tc.Name, err)
		} else {
			// 尝试将结果序列化为 JSON 字符串
			resultBytes, jsonErr := json.Marshal(result)
			if jsonErr != nil {
				content = fmt.Sprintf("Failed to serialize result for tool %s: %v", tc.Name, jsonErr)
			} else {
				content = string(resultBytes)
				// 对工具响应内容进行长度限制，防止消息过长导致API调用失败
				content = s.truncateToolResponse(content, tc.Name)
			}
		}

		// 创建工具结果消息
		toolMessage := llm.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: tc.ID,
			Name:       tc.Name,
		}
		toolMessages = append(toolMessages, toolMessage)
	}

	return toolMessages, nil
}

// getSystemPrompt 根据模式生成系统提示词
func (s *Session) getSystemPrompt() string {
	// toolDescriptions := s.buildToolDescriptions()
	toolDescriptions := ""

	switch s.config.Mode {
	case "agent":
		return s.getAgentSystemPrompt(toolDescriptions)
	default: // "chat"
		return s.getChatSystemPrompt(toolDescriptions)
	}
}

// buildToolDescriptions 构建工具描述
// func (s *Session) buildToolDescriptions() string {
// 	var descriptions []string
// 	for _, tool := range s.toolDefs {
// 		descriptions = append(descriptions, fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
// 	}
// 	return strings.Join(descriptions, "\n")
// }

// getChatSystemPrompt 普通对话模式的系统提示词
func (s *Session) getChatSystemPrompt(toolDescriptions string) string {
	thinkingPrompt := ""
	if s.config.ShowThinking {
		thinkingPrompt = `

重要：你必须在每次回答时都展示思考过程。请严格按照以下格式：

**思考过程开始**
1. 问题分析：用户问的是...
2. 思考方向：我需要考虑...
3. 解决方案：我的回答策略是...
**思考过程结束**

然后给出清晰的正式回答。`
	}

	return fmt.Sprintf(`你是一个智能的AI助手，专注于帮助用户解决问题和提供信息。

可用工具:
%s

工作特点:
- 友好、耐心、准确地回答用户问题
- 主动使用工具获取实时信息和执行操作
- 提供清晰、结构化的回答
- 在需要时展示思考过程

工具使用指导:
- 使用参数过滤减少不必要的数据量
- 系统会自动截断过长的工具响应
- 专注于用户关心的核心信息

回答风格:
- 简洁明了，直接回答用户问题
- 适当使用markdown格式提升可读性
- 必要时提供代码示例和解决方案
- 如果需要思考，可以说明推理过程%s`, toolDescriptions, thinkingPrompt)
}

// getAgentSystemPrompt 智能体模式的系统提示词
func (s *Session) getAgentSystemPrompt(toolDescriptions string) string {
	thinkingPrompt := ""
	if s.config.ShowThinking {
		thinkingPrompt = `

重要：你必须在每次回答时都展示思考过程。请严格按照以下格式：

**思考过程开始**
1. 任务理解：用户想要...
2. 分析过程：需要调用什么工具...
3. 执行计划：具体步骤...
4. 结果分析：工具返回了什么...
**思考过程结束**

然后给出用户友好的正式回答。`
	}

	return fmt.Sprintf(`你是一个自主的智能体，能够分析复杂任务并制定执行计划。

可用工具:
%s

工作模式:
1. 任务分析: 理解用户需求，识别关键要素
2. 计划制定: 将复杂任务分解为可执行的步骤
3. 自主执行: 主动调用工具，收集信息，执行操作
4. 结果整合: 综合各步骤结果，提供完整解决方案

执行特点:
- 具备强烈的目标导向性
- 主动探索和尝试不同方法
- 在遇到障碍时自主调整策略
- 持续优化执行效率
- 详细记录执行过程和思考逻辑

工具使用原则:
- 优先使用参数过滤来减少数据量（如使用match参数筛选指标）
- 当工具返回大量数据时，系统会自动截断过长内容
- 关注最重要的数据，避免一次性获取全部数据
- 根据任务需求选择合适的查询参数

输出要求:
- 清晰说明每个步骤的目的和方法
- 展示完整的问题解决过程
- 提供可操作的建议和下一步行动%s`, toolDescriptions, thinkingPrompt)
}

// ThinkingContent 思考内容结构
type ThinkingContent struct {
	Thinking string // 思考过程
	Content  string // 正式回答
}

// ExtractThinking 从响应中提取思考内容
func ExtractThinking(response string) ThinkingContent {
	const (
		thinkingStart = "**思考过程开始**"
		thinkingEnd   = "**思考过程结束**"
	)

	startIdx := strings.Index(response, thinkingStart)
	if startIdx == -1 {
		// 没有思考标记，返回原内容
		return ThinkingContent{
			Thinking: "",
			Content:  response,
		}
	}

	endIdx := strings.Index(response, thinkingEnd)
	if endIdx == -1 {
		// 只有开始标记，没有结束标记
		return ThinkingContent{
			Thinking: "",
			Content:  response,
		}
	}

	// 提取思考内容
	thinkingStartIdx := startIdx + len(thinkingStart)
	thinking := strings.TrimSpace(response[thinkingStartIdx:endIdx])

	// 提取正式内容（移除思考部分）
	before := response[:startIdx]
	after := response[endIdx+len(thinkingEnd):]
	content := strings.TrimSpace(before + after)

	return ThinkingContent{
		Thinking: thinking,
		Content:  content,
	}
}

// RemoveThinking 移除响应中的思考标记和内容
func RemoveThinking(response string) string {
	thinking := ExtractThinking(response)
	return thinking.Content
}

// SetConfig 设置会话配置（用于调试）
func (s *Session) SetConfig(config SessionConfig) {
	s.config = config
}

// SetToolDefs 设置工具定义（用于调试）
func (s *Session) SetToolDefs(toolDefs []tools.ToolDefinition) {
	s.toolDefs = toolDefs
}

// GetSystemPromptForDebug 获取系统提示词（用于调试）
func (s *Session) GetSystemPromptForDebug() string {
	return s.getSystemPrompt()
}

// truncateToolResponse 截断工具响应内容，防止消息过长
func (s *Session) truncateToolResponse(content, toolName string) string {
	const maxLength = 8000 // 设置最大长度为8000字符

	if len(content) <= maxLength {
		return content
	}

	// 截断内容并添加说明
	truncated := content[:maxLength]
	suffix := fmt.Sprintf("\n\n[注意: %s 工具响应过长已截断，原始长度: %d 字符，显示前 %d 字符]",
		toolName, len(content), maxLength)

	util.Debugw("工具响应内容已截断", map[string]interface{}{
		"tool_name":        toolName,
		"original_length":  len(content),
		"truncated_length": maxLength,
	})

	return truncated + suffix
}
