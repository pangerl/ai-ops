package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// OpenAIClient OpenAI AI 客户端实现
type OpenAIClient struct {
	httpClient *RetryableHTTPClient
	config     ModelConfig
	modelInfo  ModelInfo
}

// NewOpenAIClient 创建新的 OpenAI 客户端
func NewOpenAIClient(config ModelConfig) (*OpenAIClient, error) {
	if config.APIKey == "" {
		return nil, NewAIError(ErrCodeAPIKeyMissing, "OpenAI API key is required", nil)
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	httpClient := NewRetryableHTTPClient(baseURL, timeout, 3, time.Second)
	httpClient.SetHeader("Authorization", "Bearer "+config.APIKey)
	// httpClient.SetHeader("OpenAI-Beta", "assistants=v2")

	modelName := config.Model
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}

	// 根据模型名称设置最大令牌数
	maxTokens := 4096
	if strings.Contains(modelName, "gpt-4") {
		if strings.Contains(modelName, "32k") {
			maxTokens = 32768
		} else if strings.Contains(modelName, "turbo") {
			maxTokens = 128000
		} else {
			maxTokens = 8192
		}
	} else if strings.Contains(modelName, "gpt-3.5-turbo") {
		if strings.Contains(modelName, "16k") {
			maxTokens = 16384
		} else {
			maxTokens = 4096
		}
	}

	return &OpenAIClient{
		httpClient: httpClient,
		config:     config,
		modelInfo: ModelInfo{
			Name:         modelName,
			Type:         "openai",
			MaxTokens:    maxTokens,
			SupportTools: true,
		},
	}, nil
}

// SendMessage 发送消息并获取响应
func (c *OpenAIClient) SendMessage(ctx context.Context, message string, tools []Tool) (*Response, error) {
	request := c.buildRequest(message, tools)

	endpoint := ""

	var response OpenAIResponse
	err := c.httpClient.PostJSONWithRetry(ctx, endpoint, request, &response)
	if err != nil {
		return nil, err
	}

	return c.parseResponse(&response)
}

// GetModelInfo 获取模型信息
func (c *OpenAIClient) GetModelInfo() ModelInfo {
	return c.modelInfo
}

// buildRequest 构建 OpenAI API 请求
func (c *OpenAIClient) buildRequest(message string, tools []Tool) *OpenAIRequest {
	request := &OpenAIRequest{
		Model: c.modelInfo.Name,
		Messages: []OpenAIMessage{
			{
				Role:    "user",
				Content: message,
			},
		},
	}

	// 添加工具定义
	if len(tools) > 0 {
		request.Tools = c.convertToolsToOpenAITools(tools)
		request.ToolChoice = "auto"
	}

	return request
}

// convertToolsToOpenAITools 将工具定义转换为 OpenAI 工具格式
func (c *OpenAIClient) convertToolsToOpenAITools(tools []Tool) []OpenAITool {
	openaiTools := make([]OpenAITool, len(tools))

	for i, tool := range tools {
		openaiTools[i] = OpenAITool{
			Type: "function",
			Function: OpenAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		}
	}

	return openaiTools
}

// parseResponse 解析 OpenAI 响应
func (c *OpenAIClient) parseResponse(response *OpenAIResponse) (*Response, error) {
	if len(response.Choices) == 0 {
		return nil, NewAIError(ErrCodeInvalidResponse, "no choices in response", nil)
	}

	choice := response.Choices[0]

	result := &Response{
		Content: choice.Message.Content,
		Usage: TokenUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}

	// 解析工具调用
	if choice.Message.ToolCalls != nil {
		for _, toolCall := range choice.Message.ToolCalls {
			if toolCall.Type == "function" {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					return nil, NewAIError(ErrCodeInvalidResponse, "failed to parse tool call arguments", err)
				}

				result.ToolCalls = append(result.ToolCalls, ToolCall{
					ID:        toolCall.ID,
					Name:      toolCall.Function.Name,
					Arguments: args,
				})
			}
		}
	}

	return result, nil
}

// OpenAI API 数据结构定义

// OpenAIRequest OpenAI API 请求结构
type OpenAIRequest struct {
	Model      string          `json:"model"`
	Messages   []OpenAIMessage `json:"messages"`
	Tools      []OpenAITool    `json:"tools,omitempty"`
	ToolChoice interface{}     `json:"tool_choice,omitempty"`
}

// OpenAIMessage 消息结构
type OpenAIMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
}

// OpenAITool 工具定义
type OpenAITool struct {
	Type     string         `json:"type"`
	Function OpenAIFunction `json:"function"`
}

// OpenAIFunction 函数定义
type OpenAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OpenAIToolCall 工具调用
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall 函数调用
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIResponse OpenAI API 响应结构
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

// OpenAIChoice 选择结构
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage 使用统计
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
