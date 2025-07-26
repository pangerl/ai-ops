package tools

import (
	"ai-ops/internal/tools/plugins"
	"ai-ops/internal/util"
)

// PluginLoader 插件加载器
type PluginLoader struct {
	pluginDir string
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(pluginDir string) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
	}
}

// LoadPlugins 加载插件工具
func (pl *PluginLoader) LoadPlugins(manager ToolManager) error {
	util.Infow("开始加载插件工具", map[string]any{
		"plugin_dir": pl.pluginDir,
	})

	// 直接创建插件工具实例
	tools := []Tool{
		plugins.NewEchoTool().(*plugins.EchoTool),
		plugins.NewWeatherTool().(*plugins.WeatherTool),
	}

	registeredCount := 0
	for _, tool := range tools {
		if err := manager.RegisterTool(tool); err != nil {
			util.LogErrorWithFields(err, "注册工具失败", map[string]any{
				"tool_name": tool.Name(),
			})
		} else {
			registeredCount++
			util.Infow("成功注册工具", map[string]any{
				"tool_name": tool.Name(),
			})
		}
	}

	util.Infow("插件工具加载完成", map[string]any{
		"registered_count": registeredCount,
		"total_count":      len(tools),
	})

	return nil
}

// 保持向后兼容
type SimplePluginLoader = PluginLoader

// NewSimplePluginLoader 创建插件加载器（向后兼容）
func NewSimplePluginLoader(pluginDir string) *SimplePluginLoader {
	return NewPluginLoader(pluginDir)
}
