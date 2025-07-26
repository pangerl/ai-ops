package ai

import (
	"ai-ops/internal/tools"
	"context"
	"fmt"
	"time"
)

// AIClient 定义 AI 模型客户端接口
type AIClient interface {
	// SendMessage 发送消息并获取响应
	SendMessage(ctx context.Context, message string, history []string, toolDefs []tools.ToolDefinition) (*Response, error)

	// GetModelInfo 获取模型信息
	GetModelInfo() ModelInfo
}

// Response AI 响应结构
type Response struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     TokenUsage `json:"usage"`
}

// ToolCall 工具调用结构
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// TokenUsage 令牌使用统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	MaxTokens    int    `json:"max_tokens"`
	SupportTools bool   `json:"support_tools"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	Type    string `toml:"type"` // "gemini" 或 "openai"
	APIKey  string `toml:"api_key"`
	BaseURL string `toml:"base_url"`
	Model   string `toml:"model"`
	Timeout int    `toml:"timeout"` // 超时时间（秒）
}

// ClientManager AI 客户端管理器
type ClientManager struct {
	clients       map[string]AIClient
	defaultClient string
	configs       map[string]ModelConfig
	retryConfig   RetryConfig
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries int
	RetryDelay time.Duration
	Enabled    bool
}

// NewClientManager 创建新的客户端管理器
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]AIClient),
		configs: make(map[string]ModelConfig),
		retryConfig: RetryConfig{
			MaxRetries: 3,
			RetryDelay: time.Second,
			Enabled:    true,
		},
	}
}

// RegisterClient 注册 AI 客户端
func (cm *ClientManager) RegisterClient(name string, client AIClient, config ModelConfig) {
	cm.clients[name] = client
	cm.configs[name] = config
}

// GetClient 获取指定名称的客户端
func (cm *ClientManager) GetClient(name string) (AIClient, bool) {
	client, exists := cm.clients[name]
	return client, exists
}

// GetDefaultClient 获取默认客户端
func (cm *ClientManager) GetDefaultClient() AIClient {
	if cm.defaultClient != "" {
		if client, exists := cm.clients[cm.defaultClient]; exists {
			return client
		}
	}

	// 如果没有设置默认客户端或默认客户端不存在，返回第一个可用的客户端
	for _, client := range cm.clients {
		return client
	}

	return nil
}

// SetDefaultClient 设置默认客户端
func (cm *ClientManager) SetDefaultClient(name string) error {
	if _, exists := cm.clients[name]; !exists {
		return ErrClientNotFound
	}
	cm.defaultClient = name
	return nil
}

// ListClients 列出所有注册的客户端
func (cm *ClientManager) ListClients() []string {
	var names []string
	for name := range cm.clients {
		names = append(names, name)
	}
	return names
}

// GetConfig 获取客户端配置
func (cm *ClientManager) GetConfig(name string) (ModelConfig, bool) {
	config, exists := cm.configs[name]
	return config, exists
}

// SetRetryConfig 设置重试配置
func (cm *ClientManager) SetRetryConfig(config RetryConfig) {
	cm.retryConfig = config
}

// GetRetryConfig 获取重试配置
func (cm *ClientManager) GetRetryConfig() RetryConfig {
	return cm.retryConfig
}

// CreateClientFromConfig 根据配置创建客户端
func (cm *ClientManager) CreateClientFromConfig(name string, config ModelConfig) error {
	var client AIClient
	var err error

	// 所有模型类型都通过 OpenAI 兼容的客户端创建
	// config.Type 字段可以用于日志记录或未来的扩展，但创建逻辑是统一的
	client, err = NewOpenAIClient(config)
	if err != nil {
		return NewAIError(ErrCodeClientCreationFailed, fmt.Sprintf("failed to create client for '%s'", name), err)
	}

	cm.RegisterClient(name, client, config)
	return nil
}

// SwitchToClient 切换到指定客户端
func (cm *ClientManager) SwitchToClient(name string) error {
	if _, exists := cm.clients[name]; !exists {
		return NewAIError(ErrCodeClientNotFound, fmt.Sprintf("client not found: %s", name), nil)
	}

	cm.defaultClient = name
	return nil
}

// SendMessageWithFallback 发送消息，支持故障转移
func (cm *ClientManager) SendMessageWithFallback(ctx context.Context, message string, history []string, toolDefs []tools.ToolDefinition) (*Response, error) {
	// 首先尝试默认客户端
	defaultClient := cm.GetDefaultClient()
	if defaultClient != nil {
		response, err := cm.sendMessageWithRetry(ctx, defaultClient, message, history, toolDefs)
		if err == nil {
			return response, nil
		}

		// 如果是网络错误或超时，尝试其他客户端
		if cm.shouldFallback(err) {
			for name, client := range cm.clients {
				if name != cm.defaultClient {
					response, fallbackErr := cm.sendMessageWithRetry(ctx, client, message, history, toolDefs)
					if fallbackErr == nil {
						return response, nil
					}
				}
			}
		}

		return nil, err
	}

	return nil, NewAIError(ErrCodeClientNotFound, "no available clients", nil)
}

// sendMessageWithRetry 带重试的消息发送
func (cm *ClientManager) sendMessageWithRetry(ctx context.Context, client AIClient, message string, history []string, toolDefs []tools.ToolDefinition) (*Response, error) {
	if !cm.retryConfig.Enabled {
		return client.SendMessage(ctx, message, history, toolDefs)
	}

	var lastErr error

	for attempt := 0; attempt <= cm.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, NewAIError(ErrCodeContextCanceled, "request context canceled", ctx.Err())
			case <-time.After(cm.retryConfig.RetryDelay):
			}
		}

		response, err := client.SendMessage(ctx, message, history, toolDefs)
		if err == nil {
			return response, nil
		}

		lastErr = err

		// 检查是否应该重试
		if !cm.shouldRetry(err) {
			break
		}
	}

	return nil, lastErr
}

// shouldRetry 判断是否应该重试
func (cm *ClientManager) shouldRetry(err error) bool {
	if aiErr, ok := err.(*AIError); ok {
		switch aiErr.Code {
		case ErrCodeNetworkFailed, ErrCodeTimeout, ErrCodeRateLimited:
			return true
		default:
			return false
		}
	}
	return false
}

// shouldFallback 判断是否应该故障转移
func (cm *ClientManager) shouldFallback(err error) bool {
	if aiErr, ok := err.(*AIError); ok {
		switch aiErr.Code {
		case ErrCodeNetworkFailed, ErrCodeTimeout:
			return true
		default:
			return false
		}
	}
	return false
}

// GetClientStatus 获取客户端状态
func (cm *ClientManager) GetClientStatus(name string) (*ClientStatus, error) {
	client, exists := cm.GetClient(name)
	if !exists {
		return nil, NewAIError(ErrCodeClientNotFound, fmt.Sprintf("client not found: %s", name), nil)
	}

	config, _ := cm.GetConfig(name)
	modelInfo := client.GetModelInfo()

	// 简单的健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	healthy := true
	var lastError string

	// 发送一个简单的测试消息来检查健康状态
	_, err := client.SendMessage(ctx, "ping", nil, nil)
	if err != nil {
		healthy = false
		lastError = err.Error()
	}

	return &ClientStatus{
		Name:      name,
		Type:      config.Type,
		Model:     modelInfo.Name,
		Healthy:   healthy,
		LastError: lastError,
		IsDefault: name == cm.defaultClient,
	}, nil
}

// GetAllClientStatus 获取所有客户端状态
func (cm *ClientManager) GetAllClientStatus() map[string]*ClientStatus {
	statuses := make(map[string]*ClientStatus)

	for name := range cm.clients {
		if status, err := cm.GetClientStatus(name); err == nil {
			statuses[name] = status
		}
	}

	return statuses
}

// ClientStatus 客户端状态
type ClientStatus struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Model     string `json:"model"`
	Healthy   bool   `json:"healthy"`
	LastError string `json:"last_error,omitempty"`
	IsDefault bool   `json:"is_default"`
}

// 删除了ModelSwitcher相关代码，简化客户端管理
