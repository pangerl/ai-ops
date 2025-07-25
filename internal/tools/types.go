package tools

import (
	"context"
	"time"
)

// Tool 工具接口定义
type Tool interface {
	// Name 获取工具名称
	Name() string

	// Description 获取工具描述
	Description() string

	// Parameters 获取工具参数schema
	Parameters() map[string]interface{}

	// Execute 执行工具
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolDefinition 工具定义结构
type ToolDefinition struct {
	Name        string                 `json:"name"`        // 工具名称
	Description string                 `json:"description"` // 工具描述
	Parameters  map[string]interface{} `json:"parameters"`  // 参数schema
}

// ToolResult 工具执行结果
type ToolResult struct {
	Success       bool          `json:"success"`          // 执行是否成功
	Result        string        `json:"result,omitempty"` // 执行结果
	Error         string        `json:"error,omitempty"`  // 错误信息
	ExecutionTime time.Duration `json:"execution_time"`   // 执行时间
}

// ToolCall 工具调用结构
type ToolCall struct {
	ID        string                 `json:"id"`        // 调用ID
	Name      string                 `json:"name"`      // 工具名称
	Arguments map[string]interface{} `json:"arguments"` // 调用参数
}
