package tools

import (
	"context"
)

// PluginFactory 插件工厂函数类型
type PluginFactory func() interface{}

// Tool 工具接口定义
type Tool interface {
	// ID 返回工具的唯一标识符
	ID() string
	// Name 返回工具的名称
	Name() string
	// Type 返回工具的类型
	Type() string

	// Description 获取工具描述
	Description() string

	// Parameters 获取工具参数schema
	Parameters() map[string]any

	// Execute 执行工具
	Execute(ctx context.Context, args map[string]any) (string, error)
}

// ToolDefinition 工具定义结构
type ToolDefinition struct {
	Name        string         `json:"name"`        // 工具名称
	Description string         `json:"description"` // 工具描述
	Parameters  map[string]any `json:"parameters"`  // 参数schema
}

// ToolCall 工具调用结构
type ToolCall struct {
	ID        string         `json:"id"`        // 调用ID
	Name      string         `json:"name"`      // 工具名称
	Arguments map[string]any `json:"arguments"` // 调用参数
}
