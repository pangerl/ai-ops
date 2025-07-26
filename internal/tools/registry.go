package tools

import (
	"fmt"
	"sync"

	"ai-ops/internal/util"
)

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]Tool // 工具映射表
	mutex sync.RWMutex    // 读写锁
}

// NewToolRegistry 创建新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// RegisterTool 注册工具
func (r *ToolRegistry) RegisterTool(tool Tool) error {
	if tool == nil {
		return util.NewError(util.ErrCodeInvalidParam, "工具不能为空")
	}

	name := tool.Name()
	if name == "" {
		return util.NewError(util.ErrCodeInvalidParam, "工具名称不能为空")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 检查是否已存在同名工具
	if _, exists := r.tools[name]; exists {
		return util.NewErrorWithDetail(util.ErrCodeInvalidParam, "工具已存在",
			fmt.Sprintf("工具名称: %s", name))
	}

	r.tools[name] = tool
	util.Infow("工具注册成功", map[string]any{
		"tool_name":   name,
		"description": tool.Description(),
	})

	return nil
}

// GetTool 根据名称获取工具
func (r *ToolRegistry) GetTool(name string) (Tool, error) {
	if name == "" {
		return nil, util.NewError(util.ErrCodeInvalidParam, "工具名称不能为空")
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, util.NewErrorWithDetail(util.ErrCodeToolNotFound, "工具未找到",
			fmt.Sprintf("工具名称: %s", name))
	}

	return tool, nil
}

// GetAllTools 获取所有工具
func (r *ToolRegistry) GetAllTools() []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}

	return tools
}

// GetToolDefinitions 获取所有工具定义
func (r *ToolRegistry) GetToolDefinitions() []ToolDefinition {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	definitions := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		definitions = append(definitions, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}

	return definitions
}

// UnregisterTool 注销工具
func (r *ToolRegistry) UnregisterTool(name string) error {
	if name == "" {
		return util.NewError(util.ErrCodeInvalidParam, "工具名称不能为空")
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[name]; !exists {
		return util.NewErrorWithDetail(util.ErrCodeToolNotFound, "工具未找到",
			fmt.Sprintf("工具名称: %s", name))
	}

	delete(r.tools, name)
	util.Infow("工具注销成功", map[string]any{
		"tool_name": name,
	})

	return nil
}
