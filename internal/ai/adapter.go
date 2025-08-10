package ai

import (
	"context"
)

// AdapterCapability 定义适配器特性
type AdapterCapability string

const (
	// CapabilityChat 基础对话能力
	CapabilityChat AdapterCapability = "chat"
	// CapabilityToolCalling 工具调用能力
	CapabilityToolCalling AdapterCapability = "tool_calling"
	// CapabilityTextGeneration 文本生成能力
	CapabilityTextGeneration AdapterCapability = "text_generation"
)

// AdapterInfo 适配器信息
type AdapterInfo struct {
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	Provider        string                 `json:"provider"`
	DefaultModel    string                 `json:"default_model"`
	SupportedModels []string               `json:"supported_models"`
	Capabilities    []AdapterCapability    `json:"capabilities"`
	MaxTokens       int                    `json:"max_tokens"`
	ConfigSchema    map[string]interface{} `json:"config_schema,omitempty"`
}

// ModelAdapter 适配器接口，扩展现有的 AIClient 接口
type ModelAdapter interface {
	// 继承原有的 AIClient 接口，保持向后兼容性
	AIClient

	// GetAdapterInfo 获取适配器信息
	GetAdapterInfo() AdapterInfo

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// ValidateConfig 验证配置
	ValidateConfig(config interface{}) error

	// GetMetrics 获取指标
	GetMetrics() AdapterMetrics

	// GetStatus 获取适配器状态
	GetStatus() AdapterStatus
}

// AdapterFactory 适配器工厂函数类型
type AdapterFactory func(config interface{}) (ModelAdapter, error)

// ConfigValidator 配置验证器类型
type ConfigValidator func(config interface{}) error

// AdapterMetrics 适配器指标
type AdapterMetrics struct {
	RequestCount        int64  `json:"request_count"`
	ErrorCount          int64  `json:"error_count"`
	AverageResponseTime int64  `json:"average_response_time"`
	LastRequestTime     int64  `json:"last_request_time"`
	TokensUsed          int64  `json:"tokens_used"`
	LastError           string `json:"last_error,omitempty"`
}

// AdapterStatus 适配器状态
type AdapterStatus struct {
	Name            string                 `json:"name"`
	Healthy         bool                   `json:"healthy"`
	LastHealthCheck int64                  `json:"last_health_check"`
	Metrics         AdapterMetrics         `json:"metrics"`
	Config          map[string]interface{} `json:"config,omitempty"`
	LastError       string                 `json:"last_error,omitempty"`
}
