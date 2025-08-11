package tools

import (
	"ai-ops/internal/pkg"
	"ai-ops/internal/pkg/errors"
	"ai-ops/internal/pkg/registry"
	"ai-ops/internal/services"
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

const (
	// ToolRegistryKey 是工具注册表在中央服务中的键名
	ToolRegistryKey = "tools"
)

// ToolManager 工具管理器接口
type ToolManager interface {
	// RegisterTool 注册工具
	RegisterTool(tool Tool) error

	// RegisterToolFactory 注册插件工厂函数
	RegisterToolFactory(name string, factory PluginFactory)

	// InitializePlugins 创建所有已注册插件的工具实例
	InitializePlugins()

	// GetTools 获取所有工具
	GetTools() []Tool

	// GetTool 根据名称获取工具
	GetTool(name string) (Tool, error)

	// ExecuteToolCall 执行工具调用
	ExecuteToolCall(ctx context.Context, call ToolCall) (string, error)

	// GetToolDefinitions 获取工具定义列表
	GetToolDefinitions() []ToolDefinition
}

// DefaultToolManager 默认工具管理器实现
type DefaultToolManager struct {
	registry  registry.Registry[Tool] // 工具注册表
	factories map[string]PluginFactory
	mutex     sync.RWMutex
}

// NewToolManager 创建新的工具管理器
func NewToolManager() (ToolManager, error) {
	service := services.GetRegistryService()
	instance, exists := service.Get(ToolRegistryKey)
	if !exists {
		return nil, fmt.Errorf("tool registry with key '%s' not found in central service", ToolRegistryKey)
	}

	reg, ok := instance.(registry.Registry[Tool])
	if !ok {
		return nil, fmt.Errorf("invalid type for tool registry in central service")
	}

	return &DefaultToolManager{
		registry:  reg,
		factories: make(map[string]PluginFactory),
	}, nil
}

// InitRegistry 初始化工具注册表并将其注册到中央服务
func InitRegistry() {
	service := services.GetRegistryService()
	if _, exists := service.Get(ToolRegistryKey); !exists {
		reg := registry.NewRegistry[Tool]()
		err := service.Register(ToolRegistryKey, reg)
		if err != nil {
			panic(fmt.Sprintf("failed to register tool registry: %v", err))
		}
	}
}

// RegisterTool 注册工具
func (m *DefaultToolManager) RegisterTool(tool Tool) error {
	pkg.Infow("注册工具", map[string]any{
		"tool_name": tool.ID(),
	})
	return m.registry.Register(tool)
}

// RegisterToolFactory 注册插件工厂函数
func (m *DefaultToolManager) RegisterToolFactory(name string, factory PluginFactory) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.factories[name] = factory
	pkg.Debugw("插件工厂已注册", map[string]any{
		"plugin_name": name,
	})
}

// InitializePlugins 创建所有已注册插件的工具实例
func (m *DefaultToolManager) InitializePlugins() {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for name, factory := range m.factories {
		instance := factory()

		// 使用反射检查是否实现了Tool接口
		if tool, ok := instance.(Tool); ok {
			pkg.Debugw("创建插件工具实例", map[string]any{
				"plugin_name": name,
				"tool_name":   tool.Name(),
			})
			if err := m.RegisterTool(tool); err != nil {
				pkg.Warnw("注册插件工具失败", map[string]any{
					"plugin_name": name,
					"tool_name":   tool.Name(),
					"error":       err,
				})
			}
		} else {
			pkg.Warnw("插件实例未实现Tool接口", map[string]any{
				"plugin_name": name,
				"type":        reflect.TypeOf(instance).String(),
			})
		}
	}
}

// GetTools 获取所有工具
func (m *DefaultToolManager) GetTools() []Tool {
	return m.registry.List()
}

// GetTool 根据名称获取工具
func (m *DefaultToolManager) GetTool(name string) (Tool, error) {
	tool, found := m.registry.Get(name)
	if !found {
		return nil, errors.NewToolNotFoundError(name)
	}
	return tool, nil
}

// GetToolDefinitions 获取工具定义列表
func (m *DefaultToolManager) GetToolDefinitions() []ToolDefinition {
	tools := m.registry.List()
	definitions := make([]ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		definitions = append(definitions, ToolDefinition{
			Name:        tool.ID(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return definitions
}

// ExecuteToolCall 执行工具调用
func (m *DefaultToolManager) ExecuteToolCall(ctx context.Context, call ToolCall) (string, error) {
	startTime := time.Now()

	// 记录工具调用开始
	pkg.Infow("开始执行工具调用", map[string]any{
		"tool_name": call.Name,
		"call_id":   call.ID,
		"arguments": call.Arguments,
	})

	// 获取工具
	tool, err := m.GetTool(call.Name)
	if err != nil {
		pkg.LogErrorWithFields(err, "工具获取失败", map[string]any{
			"tool_name": call.Name,
			"call_id":   call.ID,
		})
		return "", err
	}

	// 执行工具
	result, err := tool.Execute(ctx, call.Arguments)
	executionTime := time.Since(startTime)

	if err != nil {
		// 记录执行失败
		pkg.LogErrorWithFields(err, "工具执行失败", map[string]any{
			"tool_name":      call.Name,
			"call_id":        call.ID,
			"execution_time": executionTime,
		})

		// 包装错误
		wrappedErr := errors.WrapToolError(
			fmt.Sprintf("工具 %s 执行失败", call.Name), err)
		return "", wrappedErr
	}

	// 记录执行成功
	pkg.Infow("工具执行成功", map[string]any{
		"tool_name":      call.Name,
		"call_id":        call.ID,
		"execution_time": executionTime,
		"result_length":  len(result),
	})

	return result, nil
}
