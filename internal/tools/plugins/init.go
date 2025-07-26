// Package plugins 包含所有插件工具的实现
//
// 每个插件工具都应该：
// 1. 实现 tools.Tool 接口
// 2. 在 init() 函数中调用 tools.RegisterPluginFactory() 注册自己
// 3. 提供一个 New*Tool() 工厂函数
//
// 示例：
//
//	func init() {
//	    tools.RegisterPluginFactory("my_tool", NewMyTool)
//	}
package plugins

import (
	"ai-ops/internal/util"
)

func init() {
	util.Debug("插件包初始化完成")
}
