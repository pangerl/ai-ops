package sources

import (
	"context"
	"fmt"
	"sync"

	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// PluginToolSource 插件工具源实现
type PluginToolSource struct {
	name      string
	status    tools.SourceStatus
	toolCount int
	lastError string
	mutex     sync.RWMutex
}

// NewPluginToolSource 创建新的插件工具源
func NewPluginToolSource(name string) *PluginToolSource {
	return &PluginToolSource{
		name:   name,
		status: tools.SourceStatusUnknown,
		mutex:  sync.RWMutex{},
	}
}

// GetName 获取工具源名称
func (s *PluginToolSource) GetName() string {
	return s.name
}

// GetType 获取工具源类型
func (s *PluginToolSource) GetType() tools.SourceType {
	return tools.SourceTypePlugin
}

// GetStatus 获取当前状态
func (s *PluginToolSource) GetStatus() tools.SourceStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.status
}

// Initialize 初始化工具源
func (s *PluginToolSource) Initialize(ctx context.Context, config tools.ToolSourceConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.status = tools.SourceStatusInitializing
	s.lastError = ""

	util.Infow("初始化插件工具源", map[string]any{
		"source_name": s.name,
		"enabled":     config.Enabled,
	})

	if !config.Enabled {
		s.status = tools.SourceStatusDisabled
		util.Infow("插件工具源已禁用", map[string]any{
			"source_name": s.name,
		})
		return nil
	}

	// 获取当前可用的插件工具数量
	pluginTools := tools.CreatePluginTools()
	s.toolCount = len(pluginTools)

	s.status = tools.SourceStatusReady
	util.Infow("插件工具源初始化成功", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	return nil
}

// LoadTools 加载工具到工具管理器
func (s *PluginToolSource) LoadTools(ctx context.Context, manager tools.ToolManager) error {
	s.mutex.RLock()
	status := s.status
	s.mutex.RUnlock()

	if status != tools.SourceStatusReady {
		return util.WrapError(util.ErrCodeInvalidState,
			fmt.Sprintf("插件工具源未就绪: %s, 状态: %s", s.name, status), tools.ErrSourceNotReady)
	}

	util.Infow("开始加载插件工具", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	// 创建并注册所有插件工具
	pluginTools := tools.CreatePluginTools()

	var lastError error
	successCount := 0

	for _, tool := range pluginTools {
		if err := manager.RegisterTool(tool); err != nil {
			util.LogErrorWithFields(err, "注册插件工具失败", map[string]any{
				"source_name": s.name,
				"tool_name":   tool.Name(),
			})
			lastError = err
			continue
		}
		successCount++
	}

	// 更新实际加载的工具数量
	s.mutex.Lock()
	s.toolCount = successCount
	if lastError != nil {
		s.lastError = lastError.Error()
	} else {
		s.lastError = ""
	}
	s.mutex.Unlock()

	util.Infow("插件工具加载成功", map[string]any{
		"source_name":   s.name,
		"total_tools":   len(pluginTools),
		"success_count": successCount,
		"failed_count":  len(pluginTools) - successCount,
	})

	return lastError
}

// GetTools 获取工具列表（不注册到管理器）
func (s *PluginToolSource) GetTools(ctx context.Context) ([]tools.Tool, error) {
	s.mutex.RLock()
	status := s.status
	s.mutex.RUnlock()

	if status != tools.SourceStatusReady {
		return nil, util.WrapError(util.ErrCodeInvalidState,
			fmt.Sprintf("插件工具源未就绪: %s, 状态: %s", s.name, status), tools.ErrSourceNotReady)
	}

	// 创建所有插件工具实例
	pluginTools := tools.CreatePluginTools()

	util.Infow("获取插件工具列表", map[string]any{
		"source_name": s.name,
		"tool_count":  len(pluginTools),
	})

	return pluginTools, nil
}

// GetToolCount 获取工具数量
func (s *PluginToolSource) GetToolCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.toolCount
}

// Shutdown 关闭工具源
func (s *PluginToolSource) Shutdown(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	util.Infow("关闭插件工具源", map[string]any{
		"source_name": s.name,
	})

	// 插件工具源没有需要特别关闭的资源
	// 所有插件工具都是通过工厂函数创建的无状态实例

	s.status = tools.SourceStatusUnknown
	s.toolCount = 0
	s.lastError = ""

	util.Infow("插件工具源关闭完成", map[string]any{
		"source_name": s.name,
	})

	return nil
}

// Refresh 刷新工具源
func (s *PluginToolSource) Refresh(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	util.Infow("刷新插件工具源", map[string]any{
		"source_name": s.name,
	})

	// 重新获取插件工具数量
	pluginTools := tools.CreatePluginTools()
	s.toolCount = len(pluginTools)

	s.status = tools.SourceStatusReady
	s.lastError = ""

	util.Infow("插件工具源刷新成功", map[string]any{
		"source_name": s.name,
		"tool_count":  s.toolCount,
	})

	return nil
}

// GetInfo 获取工具源信息
func (s *PluginToolSource) GetInfo() tools.ToolSourceInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return tools.ToolSourceInfo{
		Name:      s.name,
		Type:      tools.SourceTypePlugin,
		Status:    s.status,
		ToolCount: s.toolCount,
		LastError: s.lastError,
	}
}
