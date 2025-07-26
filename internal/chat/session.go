package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"ai-ops/internal/ai"
	"ai-ops/internal/tools"
)

// Session 管理一个独立的对话会话
type Session struct {
	client      ai.AIClient
	toolManager tools.ToolManager
	messages    []ai.Message
	toolDefs    []tools.ToolDefinition
}

// NewSession 创建一个新的对话会话
func NewSession(client ai.AIClient, toolManager tools.ToolManager) *Session {
	return &Session{
		client:      client,
		toolManager: toolManager,
		messages:    make([]ai.Message, 0),
		toolDefs:    toolManager.GetToolDefinitions(),
	}
}

// ProcessMessage 处理用户输入并返回最终的 AI 响应
func (s *Session) ProcessMessage(ctx context.Context, userInput string) (string, error) {
	// 将用户输入添加到消息历史
	s.messages = append(s.messages, ai.Message{Role: "user", Content: userInput})

	for {
		// 发送消息到 AI
		resp, err := s.client.SendMessage(ctx, s.messages, s.toolDefs)
		if err != nil {
			// 如果出错，从历史中移除最后一条消息，以备重试
			s.messages = s.messages[:len(s.messages)-1]
			return "", fmt.Errorf("failed to send message to AI: %w", err)
		}

		// 将 AI 的响应（不含工具调用）添加到历史记录
		aiResponseMsg := ai.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		s.messages = append(s.messages, aiResponseMsg)

		// 根据 finish_reason 决定下一步操作
		switch resp.FinishReason {
		case "stop":
			// 对话完成，返回最终内容
			return resp.Content, nil
		case "tool_calls":
			// 需要调用工具
			toolResults, err := s.executeTools(ctx, resp.ToolCalls)
			if err != nil {
				return "", fmt.Errorf("failed to execute tools: %w", err)
			}
			// 将工具结果添加到历史记录中，然后继续循环
			s.messages = append(s.messages, toolResults...)
		default:
			// 未知的 finish_reason
			return "", fmt.Errorf("unexpected finish_reason: %s", resp.FinishReason)
		}
	}
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
