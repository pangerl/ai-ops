package tools

import (
	"context"
	"fmt"
	"time"

	"ai-ops/internal/util"
)

// ToolManager 工具管理器接口
type ToolManager interface {
	// RegisterTool 注册工具
	RegisterTool(tool Tool) error

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
	registry *ToolRegistry // 工具注册表
}

// NewToolManager 创建新的工具管理器
func NewToolManager() ToolManager {
	return &DefaultToolManager{
		registry: NewToolRegistry(),
	}
}

// RegisterTool 注册工具
func (m *DefaultToolManager) RegisterTool(tool Tool) error {
	return m.registry.RegisterTool(tool)
}

// GetTools 获取所有工具
func (m *DefaultToolManager) GetTools() []Tool {
	return m.registry.GetAllTools()
}

// GetTool 根据名称获取工具
func (m *DefaultToolManager) GetTool(name string) (Tool, error) {
	return m.registry.GetTool(name)
}

// GetToolDefinitions 获取工具定义列表
func (m *DefaultToolManager) GetToolDefinitions() []ToolDefinition {
	return m.registry.GetToolDefinitions()
}

// ExecuteToolCall 执行工具调用
func (m *DefaultToolManager) ExecuteToolCall(ctx context.Context, call ToolCall) (string, error) {
	startTime := time.Now()

	// 记录工具调用开始
	util.Infow("开始执行工具调用", map[string]any{
		"tool_name": call.Name,
		"call_id":   call.ID,
		"arguments": call.Arguments,
	})

	// 获取工具
	tool, err := m.registry.GetTool(call.Name)
	if err != nil {
		util.LogErrorWithFields(err, "工具获取失败", map[string]any{
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
		util.LogErrorWithFields(err, "工具执行失败", map[string]any{
			"tool_name":      call.Name,
			"call_id":        call.ID,
			"execution_time": executionTime,
		})

		// 包装错误
		wrappedErr := util.WrapError(util.ErrCodeToolExecutionFailed,
			fmt.Sprintf("工具 %s 执行失败", call.Name), err)
		return "", wrappedErr
	}

	// 记录执行成功
	util.Infow("工具执行成功", map[string]any{
		"tool_name":      call.Name,
		"call_id":        call.ID,
		"execution_time": executionTime,
		"result_length":  len(result),
	})

	return result, nil
}
