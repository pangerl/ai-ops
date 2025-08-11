// Package plugins 包含所有插件工具的实现
//
// 每个插件工具都应该：
// 1. 实现 tools.Tool 接口
// 2. 在 init() 函数中调用 tools.DefaultManager.RegisterToolFactory() 注册自己
// 3. 提供一个 New*Tool() 工厂函数
//
// 示例：
//
//	func init() {
//	    tools.DefaultManager.RegisterToolFactory("my_tool", NewMyTool)
//	}
package plugins

import (
	"ai-ops/internal/config"
	"ai-ops/internal/tools"
	"ai-ops/internal/util"
)

// init 在包导入时自动执行，注册所有可用的插件工具
func init() {
	util.Debug("正在注册插件工厂...")

	// 注册 echo 工具
	tools.DefaultManager.RegisterToolFactory("echo", NewEchoTool)

	// 注册 weather 工具
	tools.DefaultManager.RegisterToolFactory("weather", NewWeatherTool)

	// 仅在 rag.enable=true 时注册 RAG 工具
	if config.Config != nil && config.Config.RAG.Enable {
		tools.DefaultManager.RegisterToolFactory("rag", NewRAGTool)
	}

	util.Debug("插件工厂注册完成")
}
