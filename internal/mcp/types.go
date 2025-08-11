package mcp

import (
	"context"
	"time"

	"ai-ops/internal/common/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServerConfig MCP服务器配置
type MCPServerConfig struct {
	Disabled bool              `json:"disabled"`
	Timeout  int               `json:"timeout"`
	Type     string            `json:"type"`
	Command  string            `json:"command"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env,omitempty"`
}

// MCPSettings MCP配置文件结构
type MCPSettings struct {
	MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPManager MCP管理器接口
type MCPManager interface {
	// LoadSettings 加载MCP配置
	LoadSettings(configPath string) error

	// InitializeClients 初始化所有MCP客户端
	InitializeClients(ctx context.Context) error

	// GetClients 获取所有客户端
	GetClients() map[string]*mcp.ClientSession

	// GetClient 根据名称获取客户端
	GetClient(name string) (*mcp.ClientSession, bool)

	// Shutdown 关闭所有客户端
	Shutdown() error
}

// MCPErrorFactory MCP错误工厂接口
type MCPErrorFactory interface {
	// NewMCPNotConfiguredError 创建MCP未配置错误
	NewMCPNotConfiguredError(message string) *errors.AppError

	// NewMCPConnectionFailedError 创建MCP连接失败错误
	NewMCPConnectionFailedError(message string) *errors.AppError

	// NewMCPNotConnectedError 创建MCP未连接错误
	NewMCPNotConnectedError(message string) *errors.AppError

	// NewMCPToolListFailedError 创建MCP工具列表获取失败错误
	NewMCPToolListFailedError(message string) *errors.AppError

	// NewMCPToolCallFailedError 创建MCP工具调用失败错误
	NewMCPToolCallFailedError(message string) *errors.AppError
}

// mcpErrorFactory 默认MCP错误工厂实现
type mcpErrorFactory struct{}

// NewMCPNotConfiguredError 创建MCP未配置错误
func (f *mcpErrorFactory) NewMCPNotConfiguredError(message string) *errors.AppError {
	return errors.NewError(errors.ErrCodeMCPNotConfigured, message)
}

// NewMCPConnectionFailedError 创建MCP连接失败错误
func (f *mcpErrorFactory) NewMCPConnectionFailedError(message string) *errors.AppError {
	return errors.NewError(errors.ErrCodeMCPConnectionFailed, message)
}

// NewMCPNotConnectedError 创建MCP未连接错误
func (f *mcpErrorFactory) NewMCPNotConnectedError(message string) *errors.AppError {
	return errors.NewError(errors.ErrCodeMCPNotConnected, message)
}

// NewMCPToolListFailedError 创建MCP工具列表获取失败错误
func (f *mcpErrorFactory) NewMCPToolListFailedError(message string) *errors.AppError {
	return errors.NewError(errors.ErrCodeMCPToolListFailed, message)
}

// NewMCPToolCallFailedError 创建MCP工具调用失败错误
func (f *mcpErrorFactory) NewMCPToolCallFailedError(message string) *errors.AppError {
	return errors.NewError(errors.ErrCodeMCPToolCallFailed, message)
}

// 默认MCP错误工厂实例
var defaultMCPErrorFactory = &mcpErrorFactory{}

// NewMCPNotConfiguredError 创建MCP未配置错误
func NewMCPNotConfiguredError(message string) *errors.AppError {
	return defaultMCPErrorFactory.NewMCPNotConfiguredError(message)
}

// NewMCPConnectionFailedError 创建MCP连接失败错误
func NewMCPConnectionFailedError(message string) *errors.AppError {
	return defaultMCPErrorFactory.NewMCPConnectionFailedError(message)
}

// NewMCPNotConnectedError 创建MCP未连接错误
func NewMCPNotConnectedError(message string) *errors.AppError {
	return defaultMCPErrorFactory.NewMCPNotConnectedError(message)
}

// NewMCPToolListFailedError 创建MCP工具列表获取失败错误
func NewMCPToolListFailedError(message string) *errors.AppError {
	return defaultMCPErrorFactory.NewMCPToolListFailedError(message)
}

// NewMCPToolCallFailedError 创建MCP工具调用失败错误
func NewMCPToolCallFailedError(message string) *errors.AppError {
	return defaultMCPErrorFactory.NewMCPToolCallFailedError(message)
}

// MCPTool MCP工具包装器，实现tools.Tool接口
type MCPTool struct {
	serverName string
	session    *mcp.ClientSession
	toolInfo   *mcp.Tool
	timeout    time.Duration
}
