package mcp

import (
	"context"
	"path/filepath"
	"time"

	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// MCPService MCP集成服务
type MCPService struct {
	manager    MCPManager
	registry   *MCPToolRegistry
	configPath string
	timeout    time.Duration
}

// NewMCPService 创建新的MCP服务
func NewMCPService(toolManager tools.ToolManager, configPath string, timeout time.Duration) *MCPService {
	manager := NewMCPManager()
	registry := NewMCPToolRegistry(manager, toolManager, timeout)

	return &MCPService{
		manager:    manager,
		registry:   registry,
		configPath: configPath,
		timeout:    timeout,
	}
}

// Initialize 初始化MCP服务
func (s *MCPService) Initialize(ctx context.Context) error {
	util.Infow("初始化MCP服务", map[string]any{
		"config_path": s.configPath,
		"timeout":     s.timeout,
	})

	// 检查配置文件是否存在
	if !s.isConfigFileExists() {
		util.Infow("MCP配置文件不存在，跳过MCP初始化", map[string]any{
			"config_path": s.configPath,
		})
		return nil
	}

	// 加载配置
	if err := s.manager.LoadSettings(s.configPath); err != nil {
		return util.WrapError(util.ErrCodeConfigLoadFailed, "加载MCP配置失败", err)
	}

	// 初始化客户端
	if err := s.manager.InitializeClients(ctx); err != nil {
		return util.WrapError(util.ErrCodeMCPConnectionFailed, "初始化MCP客户端失败", err)
	}

	// 注册工具
	if err := s.registry.RegisterMCPTools(ctx); err != nil {
		return util.WrapError(util.ErrCodeInitializationFailed, "注册MCP工具失败", err)
	}

	util.Infow("MCP服务初始化完成", nil)
	return nil
}

// Shutdown 关闭MCP服务
func (s *MCPService) Shutdown() error {
	util.Infow("关闭MCP服务", nil)
	return s.manager.Shutdown()
}

// RefreshTools 刷新MCP工具
func (s *MCPService) RefreshTools(ctx context.Context) error {
	util.Infow("刷新MCP工具", nil)
	return s.registry.RefreshMCPTools(ctx)
}

// GetManager 获取MCP管理器
func (s *MCPService) GetManager() MCPManager {
	return s.manager
}

// GetConnectedServers 获取已连接的服务器列表
func (s *MCPService) GetConnectedServers() []string {
	sessions := s.manager.GetClients()
	servers := make([]string, 0, len(sessions))

	for serverName := range sessions {
		servers = append(servers, serverName)
	}

	return servers
}

// GetServerStatus 获取服务器状态信息
func (s *MCPService) GetServerStatus() map[string]bool {
	sessions := s.manager.GetClients()
	status := make(map[string]bool)

	for serverName := range sessions {
		status[serverName] = true // 如果存在于sessions中，则视为已连接
	}

	return status
}

// isConfigFileExists 检查配置文件是否存在
func (s *MCPService) isConfigFileExists() bool {
	if s.configPath == "" {
		return false
	}

	// 检查绝对路径
	if filepath.IsAbs(s.configPath) {
		return util.FileExists(s.configPath)
	}

	// 检查相对路径
	return util.FileExists(s.configPath)
}
