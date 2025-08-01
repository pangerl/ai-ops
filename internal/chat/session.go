package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"ai-ops/internal/ai"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// Session 管理一个独立的对话会话
type Session struct {
	client      ai.AIClient
	toolManager tools.ToolManager
	messages    []ai.Message
	toolDefs    []tools.ToolDefinition
	maxHistory  int // 最大历史记录条数
}

// NewSession 创建一个新的对话会话
func NewSession(client ai.AIClient, toolManager tools.ToolManager) *Session {
	return &Session{
		client:      client,
		toolManager: toolManager,
		messages:    make([]ai.Message, 0),
		toolDefs:    toolManager.GetToolDefinitions(),
		maxHistory:  10, // 默认保留最近10条消息
	}
}

// ProcessMessage 处理用户输入并返回最终的 AI 响应
func (s *Session) ProcessMessage(ctx context.Context, userInput string) (string, error) {
	// 标记本轮对话的起始位置
	roundStartIndex := len(s.messages)
	// 将用户输入添加到消息历史
	s.messages = append(s.messages, ai.Message{Role: "user", Content: userInput})

	for {
		s.trimHistory()
		// 发送消息到 AI
		resp, err := s.client.SendMessage(ctx, s.messages, s.toolDefs)
		if err != nil {
			// 如果出错，从历史中移除最后一条消息，以备重试
			s.messages = s.messages[:len(s.messages)-1]
			return "", fmt.Errorf("failed to send message to AI: %w", err)
		}

		// 调试：打印完整的 AI 响应
		respBytes, _ := json.Marshal(resp)
		util.Infow("收到 AI 响应", map[string]any{"response": string(respBytes)})

		// 将 AI 的响应（不含工具调用）添加到历史记录
		aiResponseMsg := ai.Message{
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
				return "", fmt.Errorf("failed to execute tools: %w", err)
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

	s.messages = make([]ai.Message, 0, s.maxHistory)
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
	newMessages := make([]ai.Message, 0, len(previousHistory)+2)
	newMessages = append(newMessages, previousHistory...)
	newMessages = append(newMessages, userMessage)
	newMessages = append(newMessages, finalAssistantMessage)

	s.messages = newMessages
	util.Infow("历史记录已整合", map[string]any{"history_size": len(s.messages)})
}

// executeTools 执行工具调用并返回结果消息
func (s *Session) executeTools(ctx context.Context, toolCalls []ai.ToolCall) ([]ai.Message, error) {
	var toolMessages []ai.Message

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
			}
		}

		// 创建工具结果消息
		toolMessage := ai.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: tc.ID,
			Name:       tc.Name,
		}
		toolMessages = append(toolMessages, toolMessage)
	}

	return toolMessages, nil
}
