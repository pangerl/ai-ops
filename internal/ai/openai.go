package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"ai-ops/internal/common/errors"
	cfg "ai-ops/internal/config"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// OpenAIClient OpenAI AI 客户端实现，实现 ModelAdapter 接口
type OpenAIClient struct {
	*BaseAdapter // 嵌入基础适配器
	httpClient   *RetryableHTTPClient
	config       cfg.ModelConfig
	modelInfo    ModelInfo
}

// NewOpenAIClient 创建新的 OpenAI 客户端（保持向后兼容）
func NewOpenAIClient(config cfg.ModelConfig) (*OpenAIClient, error) {
	return createOpenAIClient(config)
}

// NewOpenAIAdapter 创建新的 OpenAI 适配器（工厂函数）
func NewOpenAIAdapter(config interface{}) (ModelAdapter, error) {
	modelConfig, ok := config.(cfg.ModelConfig)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeInvalidConfig, "invalid config type for OpenAI adapter")
	}
	return createOpenAIClient(modelConfig)
}

// createOpenAIClient 内部函数，创建 OpenAI 客户端实例
func createOpenAIClient(modelCfg cfg.ModelConfig) (*OpenAIClient, error) {
	if modelCfg.APIKey == "" {
		return nil, errors.NewError(errors.ErrCodeAPIKeyMissing, "OpenAI API key is required")
	}

	// 规范化 base URL，支持 style 路径风格
	// 规则：
	// - raw := strings.TrimRight(modelCfg.BaseURL, "/")
	// - 若 raw 为空：保持旧行为，使用完整 Chat Completions 端点
	// - 若 raw 已包含 "/chat/completions" 或 "/responses"：视为完整 endpoint，直接使用
	// - 否则根据 style 拼接：
	//     style == "responses"（不区分大小写）→ raw + "/responses"
	//     其他（含空/未知） → raw + "/chat/completions"
	raw := strings.TrimRight(modelCfg.BaseURL, "/")
	var effectiveBaseURL string
	if raw == "" {
		effectiveBaseURL = "https://api.openai.com/v1/chat/completions"
	} else {
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "/chat/completions") || strings.Contains(lower, "/responses") {
			effectiveBaseURL = raw
		} else {
			if strings.EqualFold(modelCfg.Style, "responses") {
				effectiveBaseURL = raw + "/responses"
			} else {
				effectiveBaseURL = raw + "/chat/completions"
			}
		}
	}

	// 获取超时配置，从全局 AI 配置或默认值
	var timeout time.Duration
	if cfg.Config != nil && cfg.Config.AI.Timeout > 0 {
		timeout = time.Duration(cfg.Config.AI.Timeout) * time.Second
	} else {
		timeout = 30 * time.Second
	}

	httpClient := NewRetryableHTTPClient(effectiveBaseURL, timeout, 3, time.Second)
	httpClient.SetHeader("Authorization", "Bearer "+modelCfg.APIKey)

	modelName := modelCfg.Model
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

	// 定义 OpenAI 适配器信息
	adapterInfo := AdapterInfo{
		Name:         "OpenAI",
		Type:         "openai",
		Version:      "1.0.0",
		Description:  "OpenAI GPT 模型适配器",
		Provider:     "OpenAI",
		DefaultModel: "gpt-4o-mini",
		SupportedModels: []string{
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4",
			"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		},
		ConfigSchema: map[string]interface{}{
			"api_key": map[string]interface{}{
				"type":        "string",
				"required":    true,
				"description": "OpenAI API 密钥",
			},
			"base_url": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "https://api.openai.com",
				"description": "API 基础 URL",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "gpt-4o-mini",
				"description": "模型名称",
			},
		},
	}

	// 设置支持的能力到适配器信息中
	adapterInfo.Capabilities = []AdapterCapability{
		CapabilityChat,
		CapabilityToolCalling,
		CapabilityTextGeneration,
	}
	adapterInfo.MaxTokens = maxTokens

	// 创建基础适配器
	baseAdapter := NewBaseAdapter(adapterInfo)

	client := &OpenAIClient{
		BaseAdapter: baseAdapter,
		httpClient:  httpClient,
		config:      modelCfg,
		modelInfo: ModelInfo{
			Name:         modelName,
			Type:         "openai",
			MaxTokens:    maxTokens,
			SupportTools: true,
		},
	}

	// 初始化适配器
	if err := client.Initialize(context.Background(), modelCfg); err != nil {
		return nil, errors.WrapError(errors.ErrCodeClientCreationFailed, "failed to initialize OpenAI adapter", err)
	}
	// 默认启用提供商特定错误映射，便于统一错误语义
	client.SetErrorMapper(CreateErrorMapperForProvider("openai"))

	util.Debugw("OpenAI 适配器创建成功", map[string]interface{}{
		"model":      modelName,
		"max_tokens": maxTokens,
		"base_url":   effectiveBaseURL,
		"style":      modelCfg.Style,
	})

	return client, nil
}

// SendMessage 发送消息并获取响应
func (c *OpenAIClient) SendMessage(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	startTime := time.Now()

	request := c.buildRequest(messages, toolDefs)

	// base_url 已经包含完整的 api 请求地址，不需要传递endpoint，保持为空
	endpoint := ""

	var response OpenAIResponse
	err := c.httpClient.PostJSONWithRetry(ctx, endpoint, request, &response)

	// 计算响应时间并更新指标
	responseTime := time.Since(startTime).Milliseconds()
	var tokensUsed int64
	if err == nil && len(response.Choices) > 0 {
		tokensUsed = int64(response.Usage.TotalTokens)
	}
	c.UpdateMetrics(responseTime, err == nil, tokensUsed)

	if err != nil {
		c.RecordError(err)
		return nil, c.MapError(err)
	}

	return c.parseResponse(&response)
}

// GetModelInfo 获取模型信息
func (c *OpenAIClient) GetModelInfo() ModelInfo {
	return c.modelInfo
}

// ValidateConfig 验证 OpenAI 配置
func (c *OpenAIClient) ValidateConfig(config interface{}) error {
	modelConfig, ok := config.(cfg.ModelConfig)
	if !ok {
		return errors.NewError(errors.ErrCodeInvalidConfig, "config must be of type cfg.ModelConfig")
	}

	if modelConfig.APIKey == "" {
		return errors.NewError(errors.ErrCodeAPIKeyMissing, "API key is required")
	}

	if modelConfig.Model != "" {
		// 验证模型是否在支持列表中
		supportedModels := []string{
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4",
			"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		}

		modelSupported := false
		for _, supported := range supportedModels {
			if strings.Contains(modelConfig.Model, supported) {
				modelSupported = true
				break
			}
		}

		if !modelSupported {
			util.Debugw("使用非标准 OpenAI 模型", map[string]interface{}{
				"model": modelConfig.Model,
			})
		}
	}

	return nil
}

// HealthCheck 健康检查
func (c *OpenAIClient) HealthCheck(ctx context.Context) error {
	// 首先调用基础适配器的健康检查
	if err := c.BaseAdapter.HealthCheck(ctx); err != nil {
		return err
	}

	// OpenAI 特定的健康检查：发送一个简单的测试请求
	testMessages := []Message{
		{Role: "user", Content: "ping"},
	}

	// 创建一个较短超时的上下文用于健康检查
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.SendMessage(healthCtx, testMessages, nil)
	if err != nil {
		return errors.WrapError(errors.ErrCodeServiceUnavailable, "OpenAI service health check failed", err)
	}

	return nil
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
		return nil, errors.NewError(errors.ErrCodeInvalidResponse, "no choices in response")
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
					return nil, errors.WrapError(errors.ErrCodeInvalidResponse, "failed to parse tool call arguments", err)
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

// validateOpenAIConfig OpenAI 配置验证器
func validateOpenAIConfig(config interface{}) error {
	modelConfig, ok := config.(cfg.ModelConfig)
	if !ok {
		return errors.NewError(errors.ErrCodeInvalidConfig, "config must be of type cfg.ModelConfig")
	}

	if modelConfig.APIKey == "" {
		return errors.NewError(errors.ErrCodeAPIKeyMissing, "API key is required for OpenAI")
	}

	return nil
}

// init 函数，在包加载时注册 OpenAI 适配器
func init() {
	// 定义适配器信息
	adapterInfo := AdapterInfo{
		Name:         "OpenAI",
		Type:         "openai",
		Version:      "1.0.0",
		Description:  "OpenAI GPT 模型适配器，支持 GPT-4 和 GPT-3.5 系列模型",
		Provider:     "OpenAI",
		DefaultModel: "gpt-4o-mini",
		SupportedModels: []string{
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4",
			"gpt-3.5-turbo", "gpt-3.5-turbo-16k",
		},
		Capabilities: []AdapterCapability{
			CapabilityChat,
			CapabilityToolCalling,
			CapabilityTextGeneration,
		},
		MaxTokens: 128000, // GPT-4 Turbo 的最大值
		ConfigSchema: map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"required":    true,
				"enum":        []string{"openai"},
				"description": "适配器类型",
			},
			"api_key": map[string]interface{}{
				"type":        "string",
				"required":    true,
				"description": "OpenAI API 密钥",
			},
			"base_url": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "https://api.openai.com",
				"description": "API 基础 URL，可用于代理服务器",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "gpt-4o-mini",
				"description": "要使用的模型名称",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"required":    false,
				"default":     30,
				"minimum":     1,
				"description": "请求超时时间（秒）",
			},
		},
	}

	// 注册适配器工厂函数
	if err := RegisterAdapterFactory("openai", NewOpenAIAdapter, adapterInfo); err != nil {
		util.Errorw("注册 OpenAI 适配器失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 注册配置验证器
	if err := RegisterConfigValidator("openai", validateOpenAIConfig); err != nil {
		util.Errorw("注册 OpenAI 配置验证器失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	util.Debugw("OpenAI 适配器注册成功", map[string]interface{}{
		"type":             "openai",
		"supported_models": adapterInfo.SupportedModels,
		"capabilities":     len(adapterInfo.Capabilities),
	})
}
