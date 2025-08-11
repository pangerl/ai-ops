package ai

import (
	"ai-ops/internal/common/errors"
	cfg "ai-ops/internal/config"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
	"context"
	"fmt"
	"strings"
	"time"
)

// GeminiClient Gemini AI 客户端实现，实现 ModelAdapter 接口
type GeminiClient struct {
	*BaseAdapter // 嵌入基础适配器
	httpClient   *RetryableHTTPClient
	config       cfg.ModelConfig
	modelInfo    ModelInfo
}

// NewGeminiClient 创建新的 Gemini 客户端（保持向后兼容）
func NewGeminiClient(config cfg.ModelConfig) (*GeminiClient, error) {
	return createGeminiClient(config)
}

// NewGeminiAdapter 创建新的 Gemini 适配器（工厂函数）
func NewGeminiAdapter(config interface{}) (ModelAdapter, error) {
	modelConfig, ok := config.(cfg.ModelConfig)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeInvalidConfig, "invalid config type for Gemini adapter")
	}
	return createGeminiClient(modelConfig)
}

// createGeminiClient 内部函数，创建 Gemini 客户端实例
func createGeminiClient(modelCfg cfg.ModelConfig) (*GeminiClient, error) {
	if modelCfg.APIKey == "" {
		return nil, errors.NewError(errors.ErrCodeAPIKeyMissing, "Gemini API key is required")
	}

	baseURL := modelCfg.BaseURL
	if baseURL == "" {
		// 使用 v1beta 端点
		baseURL = "https://generativelanguage.googleapis.com/v1beta/"
	}

	// 获取超时配置，从全局 AI 配置或默认值
	var timeout time.Duration
	if cfg.Config != nil && cfg.Config.AI.Timeout > 0 {
		timeout = time.Duration(cfg.Config.AI.Timeout) * time.Second
	} else {
		timeout = 60 * time.Second // Gemini 可能需要更长的时间
	}

	httpClient := NewRetryableHTTPClient(baseURL, timeout, 3, time.Second)
	// Gemini API 使用 x-goog-api-key header 进行认证
	httpClient.SetHeader("x-goog-api-key", modelCfg.APIKey)

	modelName := modelCfg.Model
	if modelName == "" {
		modelName = "gemini-2.0-flash-exp"
	}

	maxTokens := 8192 // gemini-pro 的默认值
	if strings.Contains(modelName, "1.5") {
		maxTokens = 1048576
	} else if strings.Contains(modelName, "2.0") {
		maxTokens = 1048576
	}

	// 定义 Gemini 适配器信息
	adapterInfo := AdapterInfo{
		Name:         "Gemini",
		Type:         "gemini",
		Version:      "1.0.0",
		Description:  "Google Gemini 模型适配器",
		Provider:     "Google",
		DefaultModel: "gemini-2.0-flash-exp",
		SupportedModels: []string{
			"gemini-2.0-flash-exp", "gemini-1.5-pro", "gemini-1.5-flash",
			"gemini-1.0-pro", "gemini-pro",
		},
		ConfigSchema: map[string]interface{}{
			"api_key": map[string]interface{}{
				"type":        "string",
				"required":    true,
				"description": "Google Gemini API 密钥",
			},
			"base_url": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "https://generativelanguage.googleapis.com/v1beta/",
				"description": "API 基础 URL",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "gemini-2.0-flash-exp",
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

	client := &GeminiClient{
		BaseAdapter: baseAdapter,
		httpClient:  httpClient,
		config:      modelCfg,
		modelInfo: ModelInfo{
			Name:         modelName,
			Type:         "gemini",
			MaxTokens:    maxTokens,
			SupportTools: true,
		},
	}

	// 初始化适配器
	if err := client.Initialize(context.Background(), modelCfg); err != nil {
		return nil, errors.WrapError(errors.ErrCodeClientCreationFailed, "failed to initialize Gemini adapter", err)
	}

	// 默认启用提供商特定错误映射
	client.SetErrorMapper(CreateErrorMapperForProvider("gemini"))

	util.Debugw("Gemini 适配器创建成功", map[string]interface{}{
		"model":      modelName,
		"max_tokens": maxTokens,
		"base_url":   baseURL,
	})

	return client, nil
}

// SendMessage 发送消息并获取响应
func (c *GeminiClient) SendMessage(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	startTime := time.Now()

	request := c.buildRequest(messages, toolDefs)

	// Gemini API 端点格式为 models/MODEL_NAME:generateContent
	endpoint := fmt.Sprintf("models/%s:generateContent", c.modelInfo.Name)

	var response GeminiResponse
	err := c.httpClient.PostJSONWithRetry(ctx, endpoint, request, &response)

	// 计算响应时间并更新指标
	responseTime := time.Since(startTime).Milliseconds()
	var tokensUsed int64
	if err == nil && len(response.Candidates) > 0 {
		// Gemini 不直接返回令牌使用情况，使用估算
		tokensUsed = int64(len(fmt.Sprintf("%+v", messages)) / 4) // 粗略估算
	}
	c.UpdateMetrics(responseTime, err == nil, tokensUsed)

	if err != nil {
		c.RecordError(err)
		return nil, c.MapError(err)
	}

	return c.parseResponse(&response)
}

// GetModelInfo 获取模型信息
func (c *GeminiClient) GetModelInfo() ModelInfo {
	return c.modelInfo
}

// ValidateConfig 验证 Gemini 配置
func (c *GeminiClient) ValidateConfig(config interface{}) error {
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
			"gemini-2.0-flash-exp", "gemini-1.5-pro", "gemini-1.5-flash",
			"gemini-1.0-pro", "gemini-pro", "gemini-2.5-flash",
		}

		modelSupported := false
		for _, supported := range supportedModels {
			if strings.Contains(modelConfig.Model, supported) {
				modelSupported = true
				break
			}
		}

		if !modelSupported {
			util.Debugw("使用非标准 Gemini 模型", map[string]interface{}{
				"model": modelConfig.Model,
			})
		}
	}

	return nil
}

// HealthCheck 健康检查
func (c *GeminiClient) HealthCheck(ctx context.Context) error {
	// 首先调用基础适配器的健康检查
	if err := c.BaseAdapter.HealthCheck(ctx); err != nil {
		return err
	}

	// Gemini 特定的健康检查：发送一个简单的测试请求
	testMessages := []Message{
		{Role: "user", Content: "ping"},
	}

	// 创建一个较短超时的上下文用于健康检查
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.SendMessage(healthCtx, testMessages, nil)
	if err != nil {
		return errors.WrapError(errors.ErrCodeServiceUnavailable, "Gemini service health check failed", err)
	}

	return nil
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
			return nil, errors.WrapError(
				errors.ErrCodeInvalidResponse,
				"请求被阻止",
				fmt.Errorf("原因: %s, 安全评级: %v", response.PromptFeedback.BlockReason, response.PromptFeedback.SafetyRatings),
			)
		}
		return nil, errors.NewError(errors.ErrCodeInvalidResponse, "Gemini 响应中没有候选者")
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

// validateGeminiConfig Gemini 配置验证器
func validateGeminiConfig(config interface{}) error {
	modelConfig, ok := config.(cfg.ModelConfig)
	if !ok {
		return errors.NewError(errors.ErrCodeInvalidConfig, "config must be of type cfg.ModelConfig")
	}

	if modelConfig.APIKey == "" {
		return errors.NewError(errors.ErrCodeAPIKeyMissing, "API key is required for Gemini")
	}

	return nil
}

// init 函数，在包加载时注册 Gemini 适配器
func init() {
	// 定义适配器信息
	adapterInfo := AdapterInfo{
		Name:         "Gemini",
		Type:         "gemini",
		Version:      "1.0.0",
		Description:  "Google Gemini 模型适配器，支持 Gemini Pro 和 Flash 系列模型",
		Provider:     "Google",
		DefaultModel: "gemini-2.0-flash-exp",
		SupportedModels: []string{
			"gemini-2.0-flash-exp", "gemini-1.5-pro", "gemini-1.5-flash",
			"gemini-1.0-pro", "gemini-pro", "gemini-2.5-flash",
		},
		Capabilities: []AdapterCapability{
			CapabilityChat,
			CapabilityToolCalling,
			CapabilityTextGeneration,
		},
		MaxTokens: 1048576, // Gemini 1.5/2.0 的最大值
		ConfigSchema: map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"required":    true,
				"enum":        []string{"gemini"},
				"description": "适配器类型",
			},
			"api_key": map[string]interface{}{
				"type":        "string",
				"required":    true,
				"description": "Google Gemini API 密钥",
			},
			"base_url": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "https://generativelanguage.googleapis.com/v1beta/",
				"description": "API 基础 URL",
			},
			"model": map[string]interface{}{
				"type":        "string",
				"required":    false,
				"default":     "gemini-2.0-flash-exp",
				"description": "要使用的模型名称",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"required":    false,
				"default":     60,
				"minimum":     1,
				"description": "请求超时时间（秒）",
			},
		},
	}

	// 注册适配器工厂函数
	if err := RegisterAdapterFactory("gemini", NewGeminiAdapter, adapterInfo); err != nil {
		util.Errorw("注册 Gemini 适配器失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// 注册配置验证器
	if err := RegisterConfigValidator("gemini", validateGeminiConfig); err != nil {
		util.Errorw("注册 Gemini 配置验证器失败", map[string]interface{}{
			"error": err.Error(),
		})
	}

	util.Debugw("Gemini 适配器注册成功", map[string]interface{}{
		"type":             "gemini",
		"supported_models": adapterInfo.SupportedModels,
		"capabilities":     len(adapterInfo.Capabilities),
	})
}
