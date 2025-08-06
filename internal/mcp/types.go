package mcp

import (
	"context"
	"time"

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

// MCPTool MCP工具包装器，实现tools.Tool接口
type MCPTool struct {
	serverName string
	session    *mcp.ClientSession
	toolInfo   *mcp.Tool
	timeout    time.Duration
}
