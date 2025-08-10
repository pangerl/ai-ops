package tools

import (
	"context"
	"errors"
)

// SourceType 工具源类型
type SourceType string

const (
	// SourceTypeMCP MCP工具源
	SourceTypeMCP SourceType = "mcp"
	// SourceTypePlugin 插件工具源
	SourceTypePlugin SourceType = "plugin"
)

// SourceStatus 工具源状态
type SourceStatus string

const (
	// SourceStatusUnknown 未知状态
	SourceStatusUnknown SourceStatus = "unknown"
	// SourceStatusInitializing 初始化中
	SourceStatusInitializing SourceStatus = "initializing"
	// SourceStatusReady 就绪状态
	SourceStatusReady SourceStatus = "ready"
	// SourceStatusError 错误状态
	SourceStatusError SourceStatus = "error"
	// SourceStatusDisabled 已禁用
	SourceStatusDisabled SourceStatus = "disabled"
)

// ToolSourceConfig 工具源配置
type ToolSourceConfig struct {
	// Name 工具源名称
	Name string `json:"name"`
	// Type 工具源类型
	Type SourceType `json:"type"`
	// Enabled 是否启用
	Enabled bool `json:"enabled"`
	// Config 具体配置，根据类型不同而不同
	Config map[string]interface{} `json:"config"`
}

// ToolSourceInfo 工具源信息
type ToolSourceInfo struct {
	// Name 工具源名称
	Name string `json:"name"`
	// Type 工具源类型
	Type SourceType `json:"type"`
	// Status 当前状态
	Status SourceStatus `json:"status"`
	// ToolCount 提供的工具数量
	ToolCount int `json:"tool_count"`
	// LastError 最后一次错误
	LastError string `json:"last_error,omitempty"`
}

// ToolSource 工具源接口
type ToolSource interface {
	// GetName 获取工具源名称
	GetName() string

	// GetType 获取工具源类型
	GetType() SourceType

	// GetStatus 获取当前状态
	GetStatus() SourceStatus

	// Initialize 初始化工具源
	Initialize(ctx context.Context, config ToolSourceConfig) error

	// LoadTools 加载工具到工具管理器
	LoadTools(ctx context.Context, manager ToolManager) error

	// GetTools 获取工具列表（不注册到管理器）
	GetTools(ctx context.Context) ([]Tool, error)

	// GetToolCount 获取工具数量
	GetToolCount() int

	// Shutdown 关闭工具源
	Shutdown(ctx context.Context) error

	// Refresh 刷新工具源
	Refresh(ctx context.Context) error

	// GetInfo 获取工具源信息
	GetInfo() ToolSourceInfo
}

// ToolSourceManager 工具源管理器接口
type ToolSourceManager interface {
	// RegisterSource 注册工具源
	RegisterSource(source ToolSource) error

	// UnregisterSource 注销工具源
	UnregisterSource(name string) error

	// GetSource 获取工具源
	GetSource(name string) (ToolSource, error)

	// GetSources 获取所有工具源
	GetSources() []ToolSource

	// GetSourcesByType 根据类型获取工具源
	GetSourcesByType(sourceType SourceType) []ToolSource

	// InitializeAll 初始化所有工具源
	InitializeAll(ctx context.Context) error

	// LoadAllTools 加载所有工具源的工具
	LoadAllTools(ctx context.Context, manager ToolManager) error

	// RefreshAll 刷新所有工具源
	RefreshAll(ctx context.Context) error

	// ShutdownAll 关闭所有工具源
	ShutdownAll(ctx context.Context) error

	// GetSourceInfos 获取所有工具源信息
	GetSourceInfos() []ToolSourceInfo

	// LoadConfig 加载配置
	LoadConfig(configs []ToolSourceConfig) error
}

// 预定义错误
var (
	ErrSourceNotFound      = errors.New("工具源未找到")
	ErrSourceAlreadyExists = errors.New("工具源已存在")
	ErrSourceNotReady      = errors.New("工具源未就绪")
	ErrInvalidSourceType   = errors.New("无效的工具源类型")
	ErrSourceInitFailed    = errors.New("工具源初始化失败")
)
