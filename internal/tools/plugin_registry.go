package tools

import (
	"reflect"
	"sync"

	"ai-ops/internal/util"
)

// PluginFactory 插件工厂函数类型
type PluginFactory func() interface{}

// pluginRegistry 全局插件注册表
var pluginRegistry = &PluginRegistry{
	factories: make(map[string]PluginFactory),
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	factories map[string]PluginFactory
	mutex     sync.RWMutex
}

// RegisterPluginFactory 注册插件工厂函数
func RegisterPluginFactory(name string, factory PluginFactory) {
	pluginRegistry.mutex.Lock()
	defer pluginRegistry.mutex.Unlock()

	pluginRegistry.factories[name] = factory
	util.Debugw("插件工厂已注册", map[string]any{
		"plugin_name": name,
	})
}

// GetPluginFactory 获取插件工厂函数
func GetPluginFactory(name string) (PluginFactory, bool) {
	pluginRegistry.mutex.RLock()
	defer pluginRegistry.mutex.RUnlock()

	factory, exists := pluginRegistry.factories[name]
	return factory, exists
}

// ListPluginFactories 列出所有已注册的插件工厂
func ListPluginFactories() []string {
	pluginRegistry.mutex.RLock()
	defer pluginRegistry.mutex.RUnlock()

	names := make([]string, 0, len(pluginRegistry.factories))
	for name := range pluginRegistry.factories {
		names = append(names, name)
	}
	return names
}

// CreatePluginTools 创建所有已注册插件的工具实例
func CreatePluginTools() []Tool {
	pluginRegistry.mutex.RLock()
	defer pluginRegistry.mutex.RUnlock()

	tools := make([]Tool, 0, len(pluginRegistry.factories))
	for name, factory := range pluginRegistry.factories {
		instance := factory()

		// 使用反射检查是否实现了Tool接口
		if tool, ok := instance.(Tool); ok {
			util.Debugw("创建插件工具实例", map[string]any{
				"plugin_name": name,
				"tool_name":   tool.Name(),
			})
			tools = append(tools, tool)
		} else {
			util.Warnw("插件实例未实现Tool接口", map[string]any{
				"plugin_name": name,
				"type":        reflect.TypeOf(instance).String(),
			})
		}
	}
	return tools
}
