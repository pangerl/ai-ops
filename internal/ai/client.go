package ai

import (
	cfg "ai-ops/internal/config"
	"ai-ops/internal/tools"
	"context"
	"fmt"
	"time"
)

// Message 代表一个对话消息
type Message struct {
	Role       string     `json:"role"` // "user", "assistant", or "tool"
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"` // The name of the tool that was called
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // Only for role="tool"
}

// AIClient 定义 AI 模型客户端接口
type AIClient interface {
	// SendMessage 发送消息并获取响应
	SendMessage(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error)

	// GetModelInfo 获取模型信息
	GetModelInfo() ModelInfo
}

// Response AI 响应结构
type Response struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        TokenUsage `json:"usage"`
	FinishReason string     `json:"finish_reason"`
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

// ClientManager AI 客户端管理器
type ClientManager struct {
	clients       map[string]AIClient
	defaultClient string
	configs       map[string]cfg.ModelConfig
	retryConfig   RetryConfig
	registry      *AdapterRegistry // 适配器注册表
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
		configs: make(map[string]cfg.ModelConfig),
		retryConfig: RetryConfig{
			MaxRetries: 3,
			RetryDelay: time.Second,
			Enabled:    true,
		},
		registry: GetDefaultRegistry(), // 使用全局适配器注册表
	}
}

// RegisterClient 注册 AI 客户端
func (cm *ClientManager) RegisterClient(name string, client AIClient, config cfg.ModelConfig) {
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
func (cm *ClientManager) GetConfig(name string) (cfg.ModelConfig, bool) {
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
func (cm *ClientManager) CreateClientFromConfig(name string, config cfg.ModelConfig) error {
	// 优先使用适配器注册表创建客户端
	if cm.registry != nil && cm.registry.HasAdapterType(config.Type) {
		adapter, err := cm.registry.CreateAdapter(name, config.Type, config)
		if err != nil {
			return NewAIError(ErrCodeClientCreationFailed, fmt.Sprintf("failed to create adapter for '%s'", name), err)
		}

		// 将适配器作为 AIClient 注册
		cm.RegisterClient(name, adapter, config)
		return nil
	}

	// 回退到传统方式（保持向后兼容性）
	var client AIClient
	var err error

	switch config.Type {
	case "gemini":
		client, err = NewGeminiClient(config)
	case "openai":
		client, err = NewOpenAIClient(config)
	default:
		return NewAIError(ErrCodeModelNotSupported, fmt.Sprintf("unsupported model type: %s", config.Type), nil)
	}

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
func (cm *ClientManager) SendMessageWithFallback(ctx context.Context, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	// 首先尝试默认客户端
	defaultClient := cm.GetDefaultClient()
	if defaultClient != nil {
		response, err := cm.sendMessageWithRetry(ctx, defaultClient, messages, toolDefs)
		if err == nil {
			return response, nil
		}

		// 如果是网络错误或超时，尝试其他客户端
		if cm.shouldFallback(err) {
			for name, client := range cm.clients {
				if name != cm.defaultClient {
					response, fallbackErr := cm.sendMessageWithRetry(ctx, client, messages, toolDefs)
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
func (cm *ClientManager) sendMessageWithRetry(ctx context.Context, client AIClient, messages []Message, toolDefs []tools.ToolDefinition) (*Response, error) {
	if !cm.retryConfig.Enabled {
		return client.SendMessage(ctx, messages, toolDefs)
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

		response, err := client.SendMessage(ctx, messages, toolDefs)
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
	testMessage := []Message{{Role: "user", Content: "ping"}}
	_, err := client.SendMessage(ctx, testMessage, nil)
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

// GetAdapterRegistry 获取适配器注册表
func (cm *ClientManager) GetAdapterRegistry() *AdapterRegistry {
	return cm.registry
}

// SetAdapterRegistry 设置适配器注册表
func (cm *ClientManager) SetAdapterRegistry(registry *AdapterRegistry) {
	cm.registry = registry
}

// ListSupportedAdapterTypes 列出所有支持的适配器类型
func (cm *ClientManager) ListSupportedAdapterTypes() []string {
	if cm.registry == nil {
		return []string{}
	}
	return cm.registry.ListSupportedTypes()
}

// GetAdapterInfo 获取指定类型的适配器信息
func (cm *ClientManager) GetAdapterInfo(adapterType string) (AdapterInfo, bool) {
	if cm.registry == nil {
		return AdapterInfo{}, false
	}
	return cm.registry.GetAdapterInfo(adapterType)
}

// GetAllAdapterInfos 获取所有适配器类型信息
func (cm *ClientManager) GetAllAdapterInfos() map[string]AdapterInfo {
	if cm.registry == nil {
		return make(map[string]AdapterInfo)
	}
	return cm.registry.GetAllAdapterInfos()
}

// ValidateAdapterConfig 验证指定类型的适配器配置
func (cm *ClientManager) ValidateAdapterConfig(adapterType string, config interface{}) error {
	if cm.registry == nil {
		return NewAIError(ErrCodeInvalidConfig, "adapter registry not available", nil)
	}
	return cm.registry.ValidateConfig(adapterType, config)
}

// GetAdapterStatus 获取适配器状态（统一使用 ModelAdapter 接口）
func (cm *ClientManager) GetAdapterStatus(name string) (*AdapterStatus, error) {
	client, exists := cm.GetClient(name)
	if !exists {
		return nil, NewAIError(ErrCodeClientNotFound, fmt.Sprintf("client not found: %s", name), nil)
	}

	config, _ := cm.GetConfig(name)
	modelInfo := client.GetModelInfo()

	var metrics AdapterMetrics
	healthy := true
	lastError := ""

	// 优先走 ModelAdapter 接口，避免具体类型断言
	if adapter, ok := client.(ModelAdapter); ok {
		metrics = adapter.GetMetrics()
		ctx := context.Background()
		if err := adapter.HealthCheck(ctx); err != nil {
			healthy = false
			lastError = err.Error()
		}
	} else {
		// 回退：对非适配器客户端做简单健康检查
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		testMessages := []Message{{Role: "user", Content: "ping"}}
		if _, err := client.SendMessage(ctx, testMessages, nil); err != nil {
			healthy = false
			lastError = err.Error()
		}
	}

	return &AdapterStatus{
		Name:            name,
		Healthy:         healthy,
		LastHealthCheck: time.Now().Unix(),
		Metrics:         metrics,
		Config: map[string]interface{}{
			"type":  config.Type,
			"model": modelInfo.Name,
		},
		LastError: lastError,
	}, nil
}

// GetAllAdapterStatus 获取所有适配器状态
func (cm *ClientManager) GetAllAdapterStatus() map[string]*AdapterStatus {
	statuses := make(map[string]*AdapterStatus)

	for name := range cm.clients {
		if status, err := cm.GetAdapterStatus(name); err == nil {
			statuses[name] = status
		}
	}

	return statuses
}

// HealthCheckAll 对所有客户端执行健康检查
func (cm *ClientManager) HealthCheckAll(ctx context.Context) map[string]error {
	results := make(map[string]error)

	for name, client := range cm.clients {
		// 检查是否是 ModelAdapter
		if adapter, ok := client.(ModelAdapter); ok {
			results[name] = adapter.HealthCheck(ctx)
		} else {
			// 对于非适配器客户端，进行简单的健康检查
			testMessages := []Message{{Role: "user", Content: "ping"}}
			_, err := client.SendMessage(ctx, testMessages, nil)
			results[name] = err
		}
	}

	return results
}

// GetRegistryStats 获取注册表统计信息
func (cm *ClientManager) GetRegistryStats() RegistryStats {
	if cm.registry == nil {
		return RegistryStats{}
	}
	return cm.registry.GetStats()
}

// 删除了ModelSwitcher相关代码，简化客户端管理
