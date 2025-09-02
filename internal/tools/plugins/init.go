package plugins

import (
	"ai-ops/internal/config"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// RegisterPluginFactories 注册所有插件的工厂函数
func RegisterPluginFactories(tm tools.ToolManager) {
	util.Debugw("开始注册插件工厂", nil)

	// 核心工具：echo - 无条件注册
	tm.RegisterToolFactory("echo", NewEchoTool)
	util.Debugw("核心工具注册完成", map[string]any{
		"tool": "echo",
	})

	// 可选工具：根据配置决定是否注册
	cfg := config.GetConfig()

	// Sysinfo工具 - 系统信息获取，根据配置启用
	if cfg.Tools.Sysinfo {
		tm.RegisterToolFactory("sysinfo", NewSysInfoTool)
		util.Debugw("可选工具注册", map[string]any{
			"tool": "sysinfo",
		})
	}

	// Weather工具 - 需要API密钥，根据配置启用
	if cfg.Tools.Weather {
		tm.RegisterToolFactory("weather", NewWeatherTool)
		util.Debugw("可选工具注册", map[string]any{
			"tool": "weather",
		})
	}

	// RAG工具 - 需要RAG服务，根据配置启用
	if cfg.Tools.RAG {
		tm.RegisterToolFactory("rag", NewRAGTool)
		util.Debugw("可选工具注册", map[string]any{
			"tool": "rag",
		})
	}

	util.Debugw("插件工厂注册完成", nil)
}
