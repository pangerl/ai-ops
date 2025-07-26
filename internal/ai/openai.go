package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"ai-ops/internal/tools"
	"ai-ops/internal/util"
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
		modelName = "gpt-4o-mini"
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
func (c *OpenAIClient) SendMessage(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	request := c.buildRequest(messages, toolDefs)

	// 序列化请求体以进行日志记录
	requestBody, err := json.Marshal(request)
	if err != nil {
		util.LogErrorWithFields(err, "序列化 OpenAI 请求失败", nil)
		// 即使序列化失败，也继续尝试发送请求
	} else {
		util.Infow("发送 OpenAI 请求", map[string]any{
			"request_body": string(requestBody),
		})
	}
	// base_url 已经包含完整的 api 请求地址，不需要传递endpoint，保持为空
	endpoint := ""

	var response OpenAIResponse
	err = c.httpClient.PostJSONWithRetry(ctx, endpoint, request, &response)
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
func (c *OpenAIClient) buildRequest(messages []Message, toolDefs []tools.ToolDefinition) *OpenAIRequest {
	openaiMessages := make([]OpenAIMessage, len(messages))
	for i, msg := range messages {
		// 这部分需要根据 Message 结构转换为 OpenAIMessage
		// 这里只是一个基础的转换
		openaiMsg := OpenAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.Role == "tool" {
			openaiMsg.ToolCallID = msg.ToolCallID
			openaiMsg.Name = msg.Name
		}
		if len(msg.ToolCalls) > 0 {
			openaiMsg.ToolCalls = c.convertToolCallsToOpenAIToolCalls(msg.ToolCalls)
		}
		openaiMessages[i] = openaiMsg
	}

	request := &OpenAIRequest{
		Model:    c.modelInfo.Name,
		Messages: openaiMessages,
	}

	// 添加工具定义
	if len(toolDefs) > 0 {
		request.Tools = c.convertToolsToOpenAITools(toolDefs)
		request.ToolChoice = "auto"
	}

	return request
}

// convertToolsToOpenAITools 将工具定义转换为 OpenAI 工具格式
func (c *OpenAIClient) convertToolsToOpenAITools(toolDefs []tools.ToolDefinition) []OpenAITool {
	openaiTools := make([]OpenAITool, len(toolDefs))

	for i, tool := range toolDefs {
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

// convertToolCallsToOpenAIToolCalls 将工具调用转换为 OpenAI 工具调用格式
func (c *OpenAIClient) convertToolCallsToOpenAIToolCalls(toolCalls []ToolCall) []OpenAIToolCall {
	openaiToolCalls := make([]OpenAIToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		argsBytes, _ := json.Marshal(tc.Arguments)
		openaiToolCalls[i] = OpenAIToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: OpenAIFunctionCall{
				Name:      tc.Name,
				Arguments: string(argsBytes),
			},
		}
	}
	return openaiToolCalls
}

// parseResponse 解析 OpenAI 响应
func (c *OpenAIClient) parseResponse(response *OpenAIResponse) (*Response, error) {
	if len(response.Choices) == 0 {
		return nil, NewAIError(ErrCodeInvalidResponse, "no choices in response", nil)
	}

	choice := response.Choices[0]

	result := &Response{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
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
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
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
