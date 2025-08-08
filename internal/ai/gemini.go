package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-ops/internal/tools"
)

// GeminiClient Gemini AI 客户端实现
type GeminiClient struct {
	httpClient *RetryableHTTPClient
	config     ModelConfig
	modelInfo  ModelInfo
}

// NewGeminiClient 创建新的 Gemini 客户端
func NewGeminiClient(config ModelConfig) (*GeminiClient, error) {
	if config.APIKey == "" {
		return nil, NewAIError(ErrCodeAPIKeyMissing, "Gemini API key is required", nil)
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		// 使用 v1beta 端点
		baseURL = "https://generativelanguage.googleapis.com/v1beta/"
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second // Gemini 可能需要更长的时间
	}

	httpClient := NewRetryableHTTPClient(baseURL, timeout, 3, time.Second)
	// Gemini API 使用 x-goog-api-key header 进行认证
	httpClient.SetHeader("x-goog-api-key", config.APIKey)

	modelName := config.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	maxTokens := 8192 // gemini-pro 的默认值
	if strings.Contains(modelName, "1.5") {
		maxTokens = 1048576
	}

	return &GeminiClient{
		httpClient: httpClient,
		config:     config,
		modelInfo: ModelInfo{
			Name:         modelName,
			Type:         "gemini",
			MaxTokens:    maxTokens,
			SupportTools: true,
		},
	}, nil
}

// SendMessage 发送消息并获取响应
func (c *GeminiClient) SendMessage(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	request := c.buildRequest(messages, toolDefs)

	// Gemini API 端点格式为 models/MODEL_NAME:generateContent
	endpoint := fmt.Sprintf("models/%s:generateContent", c.modelInfo.Name)

	var response GeminiResponse
	err := c.httpClient.PostJSONWithRetry(ctx, endpoint, request, &response)
	if err != nil {
		return nil, err
	}

	return c.parseResponse(&response)
}

// GetModelInfo 获取模型信息
func (c *GeminiClient) GetModelInfo() ModelInfo {
	return c.modelInfo
}

// buildRequest 构建 Gemini API 请求
func (c *GeminiClient) buildRequest(messages []Message, toolDefs []tools.ToolDefinition) *GeminiRequest {
	contents := make([]GeminiContent, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == "tool" {
			// 这是来自工具调用的响应
			contents = append(contents, GeminiContent{
				Role: "tool",
				Parts: []GeminiPart{
					{
						FunctionResponse: &GeminiFunctionResponse{
							Name: msg.Name, // 被调用的工具名称
							Response: map[string]interface{}{
								"content": msg.Content, // 来自工具的结果
							},
						},
					},
				},
			})
		} else {
			// 处理 "user" 和 "assistant" 角色
			var role string
			if msg.Role == "assistant" {
				role = "model"
			} else {
				role = msg.Role
			}

			parts := []GeminiPart{}
			if msg.Content != "" {
				parts = append(parts, GeminiPart{Text: msg.Content})
			}

			// 如果历史记录中的 assistant 消息包含工具调用，则表示它们
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					parts = append(parts, GeminiPart{
						FunctionCall: &GeminiFunctionCall{
							Name: tc.Name,
							Args: tc.Arguments,
						},
					})
				}
			}

			if len(parts) > 0 {
				contents = append(contents, GeminiContent{
					Role:  role,
					Parts: parts,
				})
			}
		}
	}

	req := &GeminiRequest{
		Contents: contents,
	}

	if len(toolDefs) > 0 {
		req.Tools = c.convertToolsToGeminiTools(toolDefs)
	}

	return req
}

// convertToolsToGeminiTools 将工具定义转换为 Gemini 的格式
func (c *GeminiClient) convertToolsToGeminiTools(toolDefs []tools.ToolDefinition) []GeminiTool {
	functions := make([]GeminiFunctionDeclaration, len(toolDefs))
	for i, tool := range toolDefs {
		functions[i] = GeminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  tool.Parameters,
		}
	}
	return []GeminiTool{{FunctionDeclarations: functions}}
}

// parseResponse 解析 Gemini 响应
func (c *GeminiClient) parseResponse(response *GeminiResponse) (*Response, error) {
	if len(response.Candidates) == 0 {
		if response.PromptFeedback != nil && response.PromptFeedback.BlockReason != "" {
			return nil, NewAIErrorWithDetails(ErrCodeInvalidResponse,
				"请求被阻止",
				fmt.Sprintf("原因: %s, 安全评级: %v", response.PromptFeedback.BlockReason, response.PromptFeedback.SafetyRatings),
				nil)
		}
		return nil, NewAIError(ErrCodeInvalidResponse, "Gemini 响应中没有候选者", nil)
	}

	candidate := response.Candidates[0]
	result := &Response{
		FinishReason: candidate.FinishReason,
		// Gemini API 在标准响应中不提供令牌使用情况。
		// 它通过单独的 countTokens API 提供。
		// 我们暂时将其保留为零。
		Usage: TokenUsage{},
	}

	if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				result.Content += part.Text
			}
			if part.FunctionCall != nil {
				result.ToolCalls = append(result.ToolCalls, ToolCall{
					// Gemini 不像 OpenAI 那样提供工具调用 ID。
					// 我们可以生成一个或使用函数名。我们现在创建一个合成 ID。
					ID:        fmt.Sprintf("call_%s_%d", part.FunctionCall.Name, time.Now().UnixNano()),
					Name:      part.FunctionCall.Name,
					Arguments: part.FunctionCall.Args,
				})
			}
		}
	}

	return result, nil
}

// Gemini API 数据结构

type GeminiRequest struct {
	Contents         []GeminiContent `json:"contents"`
	Tools            []GeminiTool    `json:"tools,omitempty"`
	GenerationConfig *struct{}       `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *GeminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type GeminiFunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

type GeminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type GeminiResponse struct {
	Candidates     []GeminiCandidate `json:"candidates"`
	PromptFeedback *PromptFeedback   `json:"promptFeedback,omitempty"`
}

type GeminiCandidate struct {
	Content      *GeminiContent `json:"content"`
	FinishReason string         `json:"finishReason"`
}

type PromptFeedback struct {
	BlockReason   string         `json:"blockReason,omitempty"`
	SafetyRatings []SafetyRating `json:"safetyRatings,omitempty"`
}

type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}
