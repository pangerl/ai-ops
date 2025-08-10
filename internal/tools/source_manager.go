package tools

import (
	"context"
	"fmt"
	"sync"

	"ai-ops/internal/util"
)

// DefaultToolSourceManager 默认工具源管理器实现
type DefaultToolSourceManager struct {
	sources map[string]ToolSource // 工具源映射表
	mutex   sync.RWMutex          // 读写锁
}

// NewToolSourceManager 创建新的工具源管理器
func NewToolSourceManager() ToolSourceManager {
	return &DefaultToolSourceManager{
		sources: make(map[string]ToolSource),
	}
}

// RegisterSource 注册工具源
func (m *DefaultToolSourceManager) RegisterSource(source ToolSource) error {
	if source == nil {
		return util.NewError(util.ErrCodeInvalidParam, "工具源不能为空")
	}

	name := source.GetName()
	if name == "" {
		return util.NewError(util.ErrCodeInvalidParam, "工具源名称不能为空")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查是否已存在同名工具源
	if _, exists := m.sources[name]; exists {
		return util.WrapError(util.ErrCodeInvalidParam,
			fmt.Sprintf("工具源已存在: %s", name), ErrSourceAlreadyExists)
	}

	m.sources[name] = source
	util.Infow("工具源注册成功", map[string]any{
		"source_name": name,
		"source_type": source.GetType(),
	})

	return nil
}

// UnregisterSource 注销工具源
func (m *DefaultToolSourceManager) UnregisterSource(name string) error {
	if name == "" {
		return util.NewError(util.ErrCodeInvalidParam, "工具源名称不能为空")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	source, exists := m.sources[name]
	if !exists {
		return util.WrapError(util.ErrCodeToolNotFound,
			fmt.Sprintf("工具源未找到: %s", name), ErrSourceNotFound)
	}

	// 尝试关闭工具源
	if err := source.Shutdown(context.Background()); err != nil {
		util.LogErrorWithFields(err, "关闭工具源失败", map[string]any{
			"source_name": name,
		})
	}

	delete(m.sources, name)
	util.Infow("工具源注销成功", map[string]any{
		"source_name": name,
	})

	return nil
}

// GetSource 获取工具源
func (m *DefaultToolSourceManager) GetSource(name string) (ToolSource, error) {
	if name == "" {
		return nil, util.NewError(util.ErrCodeInvalidParam, "工具源名称不能为空")
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	source, exists := m.sources[name]
	if !exists {
		return nil, util.WrapError(util.ErrCodeToolNotFound,
			fmt.Sprintf("工具源未找到: %s", name), ErrSourceNotFound)
	}

	return source, nil
}

// GetSources 获取所有工具源
func (m *DefaultToolSourceManager) GetSources() []ToolSource {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sources := make([]ToolSource, 0, len(m.sources))
	for _, source := range m.sources {
		sources = append(sources, source)
	}

	return sources
}

// GetSourcesByType 根据类型获取工具源
func (m *DefaultToolSourceManager) GetSourcesByType(sourceType SourceType) []ToolSource {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sources := make([]ToolSource, 0)
	for _, source := range m.sources {
		if source.GetType() == sourceType {
			sources = append(sources, source)
		}
	}

	return sources
}

// InitializeAll 初始化所有工具源
func (m *DefaultToolSourceManager) InitializeAll(ctx context.Context) error {
	m.mutex.RLock()
	sources := make([]ToolSource, 0, len(m.sources))
	for _, source := range m.sources {
		sources = append(sources, source)
	}
	m.mutex.RUnlock()

	util.Infow("开始初始化所有工具源", map[string]any{
		"source_count": len(sources),
	})

	var lastError error
	successCount := 0

	for _, source := range sources {
		if source.GetStatus() == SourceStatusDisabled {
			util.Infow("跳过已禁用的工具源", map[string]any{
				"source_name": source.GetName(),
			})
			continue
		}

		// 这里需要从配置中获取具体的配置信息
		// 暂时使用空配置，具体实现时会通过LoadConfig方法设置
		config := ToolSourceConfig{
			Name:    source.GetName(),
			Type:    source.GetType(),
			Enabled: true,
			Config:  make(map[string]interface{}),
		}

		if err := source.Initialize(ctx, config); err != nil {
			util.LogErrorWithFields(err, "工具源初始化失败", map[string]any{
				"source_name": source.GetName(),
				"source_type": source.GetType(),
			})
			lastError = err
			continue
		}

		successCount++
		util.Infow("工具源初始化成功", map[string]any{
			"source_name": source.GetName(),
			"source_type": source.GetType(),
		})
	}

	util.Infow("工具源初始化完成", map[string]any{
		"total_count":   len(sources),
		"success_count": successCount,
		"failed_count":  len(sources) - successCount,
	})

	return lastError
}

// LoadAllTools 加载所有工具源的工具
func (m *DefaultToolSourceManager) LoadAllTools(ctx context.Context, manager ToolManager) error {
	sources := m.GetSources()

	util.Infow("开始加载所有工具源的工具", map[string]any{
		"source_count": len(sources),
	})

	var lastError error
	totalTools := 0
	successSources := 0

	for _, source := range sources {
		if source.GetStatus() != SourceStatusReady {
			util.Warnw("跳过未就绪的工具源", map[string]any{
				"source_name":   source.GetName(),
				"source_status": source.GetStatus(),
			})
			continue
		}

		if err := source.LoadTools(ctx, manager); err != nil {
			util.LogErrorWithFields(err, "加载工具源工具失败", map[string]any{
				"source_name": source.GetName(),
				"source_type": source.GetType(),
			})
			lastError = err
			continue
		}

		toolCount := source.GetToolCount()
		totalTools += toolCount
		successSources++

		util.Infow("工具源工具加载成功", map[string]any{
			"source_name": source.GetName(),
			"tool_count":  toolCount,
		})
	}

	util.Infow("工具源工具加载完成", map[string]any{
		"total_sources":   len(sources),
		"success_sources": successSources,
		"total_tools":     totalTools,
	})

	return lastError
}

// RefreshAll 刷新所有工具源
func (m *DefaultToolSourceManager) RefreshAll(ctx context.Context) error {
	sources := m.GetSources()

	util.Infow("开始刷新所有工具源", map[string]any{
		"source_count": len(sources),
	})

	var lastError error
	successCount := 0

	for _, source := range sources {
		if err := source.Refresh(ctx); err != nil {
			util.LogErrorWithFields(err, "刷新工具源失败", map[string]any{
				"source_name": source.GetName(),
			})
			lastError = err
			continue
		}

		successCount++
		util.Infow("工具源刷新成功", map[string]any{
			"source_name": source.GetName(),
		})
	}

	util.Infow("工具源刷新完成", map[string]any{
		"total_count":   len(sources),
		"success_count": successCount,
		"failed_count":  len(sources) - successCount,
	})

	return lastError
}

// ShutdownAll 关闭所有工具源
func (m *DefaultToolSourceManager) ShutdownAll(ctx context.Context) error {
	sources := m.GetSources()

	util.Infow("开始关闭所有工具源", map[string]any{
		"source_count": len(sources),
	})

	var lastError error

	for _, source := range sources {
		if err := source.Shutdown(ctx); err != nil {
			util.LogErrorWithFields(err, "关闭工具源失败", map[string]any{
				"source_name": source.GetName(),
			})
			lastError = err
			continue
		}

		util.Infow("工具源关闭成功", map[string]any{
			"source_name": source.GetName(),
		})
	}

	util.Infow("所有工具源关闭完成", nil)
	return lastError
}

// GetSourceInfos 获取所有工具源信息
func (m *DefaultToolSourceManager) GetSourceInfos() []ToolSourceInfo {
	sources := m.GetSources()
	infos := make([]ToolSourceInfo, len(sources))

	for i, source := range sources {
		infos[i] = source.GetInfo()
	}

	return infos
}

// LoadConfig 加载配置
func (m *DefaultToolSourceManager) LoadConfig(configs []ToolSourceConfig) error {
	util.Infow("加载工具源配置", map[string]any{
		"config_count": len(configs),
	})

	// 这里可以根据配置创建和注册工具源
	// 由于具体的工具源实现还未完成，这里暂时只记录配置
	// 实际实现时，会根据配置类型创建对应的工具源实例

	for _, config := range configs {
		util.Debugw("工具源配置", map[string]any{
			"name":    config.Name,
			"type":    config.Type,
			"enabled": config.Enabled,
		})
	}

	util.Infow("工具源配置加载完成", nil)
	return nil
}
