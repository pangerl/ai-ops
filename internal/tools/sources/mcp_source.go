package sources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"ai-ops/internal/mcp"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// MCPToolSource MCP工具源实现
type MCPToolSource struct {
	name         string
	status       tools.SourceStatus
	manager      mcp.MCPManager
	toolRegistry *mcp.MCPToolRegistry
	toolCount    int
	lastError    string
	mutex        sync.RWMutex
}

// NewMCPToolSource 创建新的MCP工具源
func NewMCPToolSource(name string, manager mcp.MCPManager) *MCPToolSource {
	return &MCPToolSource{
		name:    name,
		status:  tools.SourceStatusUnknown,
		manager: manager,
		mutex:   sync.RWMutex{},
	}
}

// GetName 获取工具源名称
func (s *MCPToolSource) GetName() string {
	return s.name
}

// GetType 获取工具源类型
func (s *MCPToolSource) GetType() tools.SourceType {
	return tools.SourceTypeMCP
}

// GetStatus 获取当前状态
func (s *MCPToolSource) GetStatus() tools.SourceStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.status
}

// Initialize 初始化工具源
func (s *MCPToolSource) Initialize(ctx context.Context, config tools.ToolSourceConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.status = tools.SourceStatusInitializing
	s.lastError = ""

	util.Infow("初始化MCP工具源", map[string]any{
		"source_name": s.name,
		"enabled":     config.Enabled,
	})

	if !config.Enabled {
		s.status = tools.SourceStatusDisabled
		util.Infow("MCP工具源已禁用", map[string]any{
			"source_name": s.name,
		})
		return nil
	}

	// 从配置中获取MCP设置路径
	configPath, ok := config.Config["config_path"].(string)
	if !ok || configPath == "" {
		// 使用默认路径
		configPath = "mcp_settings.json"
	}

	// 加载MCP设置
	if err := s.manager.LoadSettings(configPath); err != nil {
		s.status = tools.SourceStatusError
		s.lastError = err.Error()
		return util.WrapError(util.ErrCodeConfigLoadFailed,
			fmt.Sprintf("加载MCP配置失败: %s", configPath), err)
	}

	// 初始化MCP客户端
	if err := s.manager.InitializeClients(ctx); err != nil {
		s.status = tools.SourceStatusError
		s.lastError = err.Error()
		return util.WrapError(util.ErrCodeMCPConnectionFailed, "初始化MCP客户端失败", err)
	}

	// 创建工具注册器
	timeout := 30 * time.Second
	if timeoutVal, ok := config.Config["timeout"].(int); ok && timeoutVal > 0 {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	// 创建一个临时的工具管理器来计算工具数量
	tempManager := tools.NewToolManager()
	s.toolRegistry = mcp.NewMCPToolRegistry(s.manager, tempManager, timeout)

	// 注册MCP工具到临时管理器来计算数量
	if err := s.toolRegistry.RegisterMCPTools(ctx); err != nil {
		util.LogErrorWithFields(err, "注册MCP工具失败", map[string]any{
			"source_name": s.name,
		})
		// 不设为错误状态，因为部分工具注册失败是可以接受的
	}

	// 获取工具数量
	s.toolCount = len(tempManager.GetTools())

	s.status = tools.SourceStatusReady
	util.Infow("MCP工具源初始化成功", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	return nil
}

// LoadTools 加载工具到工具管理器
func (s *MCPToolSource) LoadTools(ctx context.Context, manager tools.ToolManager) error {
	s.mutex.RLock()
	status := s.status
	s.mutex.RUnlock()

	if status != tools.SourceStatusReady {
		return util.WrapError(util.ErrCodeInvalidState,
			fmt.Sprintf("MCP工具源未就绪: %s, 状态: %s", s.name, status), tools.ErrSourceNotReady)
	}

	if s.toolRegistry == nil {
		return util.NewError(util.ErrCodeInvalidState, "MCP工具注册器未初始化")
	}

	util.Infow("开始加载MCP工具", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	// 重新创建工具注册器，使用真正的工具管理器
	timeout := 30 * time.Second
	s.toolRegistry = mcp.NewMCPToolRegistry(s.manager, manager, timeout)

	// 注册MCP工具
	if err := s.toolRegistry.RegisterMCPTools(ctx); err != nil {
		s.mutex.Lock()
		s.lastError = err.Error()
		s.mutex.Unlock()
		return util.WrapError(util.ErrCodeToolRegistrationFailed, "注册MCP工具失败", err)
	}

	util.Infow("MCP工具加载成功", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	return nil
}

// GetTools 获取工具列表（不注册到管理器）
func (s *MCPToolSource) GetTools(ctx context.Context) ([]tools.Tool, error) {
	s.mutex.RLock()
	status := s.status
	s.mutex.RUnlock()

	if status != tools.SourceStatusReady {
		return nil, util.WrapError(util.ErrCodeInvalidState,
			fmt.Sprintf("MCP工具源未就绪: %s, 状态: %s", s.name, status), tools.ErrSourceNotReady)
	}

	// 创建临时工具管理器
	tempManager := tools.NewToolManager()
	timeout := 30 * time.Second
	tempRegistry := mcp.NewMCPToolRegistry(s.manager, tempManager, timeout)

	// 注册工具到临时管理器
	if err := tempRegistry.RegisterMCPTools(ctx); err != nil {
		return nil, util.WrapError(util.ErrCodeToolRegistrationFailed, "获取MCP工具失败", err)
	}

	return tempManager.GetTools(), nil
}

// GetToolCount 获取工具数量
func (s *MCPToolSource) GetToolCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.toolCount
}

// Shutdown 关闭工具源
func (s *MCPToolSource) Shutdown(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	util.Infow("关闭MCP工具源", map[string]any{
		"source_name": s.name,
	})

	var lastError error

	// 关闭MCP管理器
	if s.manager != nil {
		if err := s.manager.Shutdown(); err != nil {
			util.LogErrorWithFields(err, "关闭MCP管理器失败", map[string]any{
				"source_name": s.name,
			})
			lastError = err
		}
	}

	s.status = tools.SourceStatusUnknown
	s.toolRegistry = nil
	s.toolCount = 0

	util.Infow("MCP工具源关闭完成", map[string]any{
		"source_name": s.name,
	})

	return lastError
}

// Refresh 刷新工具源
func (s *MCPToolSource) Refresh(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	util.Infow("刷新MCP工具源", map[string]any{
		"source_name": s.name,
	})

	if s.toolRegistry == nil {
		return util.NewError(util.ErrCodeInvalidState, "MCP工具注册器未初始化")
	}

	// 重新初始化客户端
	if err := s.manager.InitializeClients(ctx); err != nil {
		s.status = tools.SourceStatusError
		s.lastError = err.Error()
		return util.WrapError(util.ErrCodeMCPConnectionFailed, "重新初始化MCP客户端失败", err)
	}

	// 刷新工具注册
	if err := s.toolRegistry.RefreshMCPTools(ctx); err != nil {
		s.lastError = err.Error()
		return util.WrapError(util.ErrCodeToolRegistrationFailed, "刷新MCP工具失败", err)
	}

	// 重新计算工具数量
	tempManager := tools.NewToolManager()
	timeout := 30 * time.Second
	tempRegistry := mcp.NewMCPToolRegistry(s.manager, tempManager, timeout)

	if err := tempRegistry.RegisterMCPTools(ctx); err == nil {
		s.toolCount = len(tempManager.GetTools())
	}

	s.status = tools.SourceStatusReady
	s.lastError = ""

	util.Infow("MCP工具源刷新成功", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	return nil
}

// GetInfo 获取工具源信息
func (s *MCPToolSource) GetInfo() tools.ToolSourceInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return tools.ToolSourceInfo{
		Name:      s.name,
		Type:      tools.SourceTypeMCP,
		Status:    s.status,
		ToolCount: s.toolCount,
		LastError: s.lastError,
	}
}
