package plugins

import (
	"ai-ops/internal/config"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// RegisterPluginFactories 注册所有插件的工厂函数
func RegisterPluginFactories(tm tools.ToolManager) {
	util.Debugw("开始注册所有插件工厂", nil)

	// 注册 EchoTool
	tm.RegisterToolFactory("echo", NewEchoTool)

	// 注册 WeatherTool
	// 注意：这里可以添加配置检查，例如，如果缺少API密钥则不注册
	tm.RegisterToolFactory("weather", NewWeatherTool)

	// 注册 RAGTool
	// 注意：这里可以添加配置检查，例如，如果RAG未启用则不注册
	if config.GetConfig().RAG.Enable {
		tm.RegisterToolFactory("rag", NewRAGTool)
	}

	// 注册系统信息工具
	tm.RegisterToolFactory("sysinfo", NewSysInfoTool)

	util.Debugw("所有插件工厂注册完成", nil)
}
