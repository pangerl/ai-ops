package tools

import (
	"testing"

	"ai-ops/internal/util"
)

func TestNewToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	if registry == nil {
		t.Error("注册表不应该为空")
		return
	}

	if registry.Count() != 0 {
		t.Errorf("新注册表应该为空，实际工具数量: %d", registry.Count())
	}
}

func TestToolRegistry_RegisterTool(t *testing.T) {
	registry := NewToolRegistry()
	tool := createMockTool("test_tool", "测试工具")

	// 测试正常注册
	err := registry.RegisterTool(tool)
	if err != nil {
		t.Errorf("注册工具时发生错误: %v", err)
	}

	if registry.Count() != 1 {
		t.Errorf("期望工具数量为1，实际为: %d", registry.Count())
	}

	// 测试重复注册
	err = registry.RegisterTool(tool)
	if err == nil {
		t.Error("重复注册应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeInvalidParam) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeInvalidParam, util.GetErrorCode(err))
	}
}

func TestToolRegistry_RegisterTool_NilTool(t *testing.T) {
	registry := NewToolRegistry()

	err := registry.RegisterTool(nil)
	if err == nil {
		t.Error("注册空工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeInvalidParam) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeInvalidParam, util.GetErrorCode(err))
	}
}

func TestToolRegistry_RegisterTool_EmptyName(t *testing.T) {
	registry := NewToolRegistry()
	tool := createMockTool("", "测试工具")

	err := registry.RegisterTool(tool)
	if err == nil {
		t.Error("注册空名称工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeInvalidParam) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeInvalidParam, util.GetErrorCode(err))
	}
}

func TestToolRegistry_GetTool(t *testing.T) {
	registry := NewToolRegistry()
	originalTool := createMockTool("test_tool", "测试工具")

	// 先注册工具
	err := registry.RegisterTool(originalTool)
	if err != nil {
		t.Errorf("注册工具时发生错误: %v", err)
	}

	// 测试获取存在的工具
	tool, err := registry.GetTool("test_tool")
	if err != nil {
		t.Errorf("获取工具时发生错误: %v", err)
	}

	if tool.Name() != "test_tool" {
		t.Errorf("期望工具名称为 'test_tool'，实际为: %s", tool.Name())
	}

	// 测试获取不存在的工具
	_, err = registry.GetTool("nonexistent_tool")
	if err == nil {
		t.Error("获取不存在的工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeToolNotFound) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeToolNotFound, util.GetErrorCode(err))
	}
}

func TestToolRegistry_GetTool_EmptyName(t *testing.T) {
	registry := NewToolRegistry()

	_, err := registry.GetTool("")
	if err == nil {
		t.Error("获取空名称工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeInvalidParam) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeInvalidParam, util.GetErrorCode(err))
	}
}

func TestToolRegistry_GetAllTools(t *testing.T) {
	registry := NewToolRegistry()

	// 测试空注册表
	tools := registry.GetAllTools()
	if len(tools) != 0 {
		t.Errorf("空注册表应该返回空切片，实际长度: %d", len(tools))
	}

	// 注册多个工具
	tool1 := createMockTool("tool1", "工具1")
	tool2 := createMockTool("tool2", "工具2")

	registry.RegisterTool(tool1)
	registry.RegisterTool(tool2)

	tools = registry.GetAllTools()
	if len(tools) != 2 {
		t.Errorf("期望工具数量为2，实际为: %d", len(tools))
	}

	// 验证工具名称
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	if !names["tool1"] || !names["tool2"] {
		t.Error("返回的工具列表不完整")
	}
}

func TestToolRegistry_GetToolDefinitions(t *testing.T) {
	registry := NewToolRegistry()

	// 注册工具
	tool := createMockTool("test_tool", "测试工具")
	registry.RegisterTool(tool)

	definitions := registry.GetToolDefinitions()
	if len(definitions) != 1 {
		t.Errorf("期望定义数量为1，实际为: %d", len(definitions))
	}

	def := definitions[0]
	if def.Name != "test_tool" {
		t.Errorf("期望定义名称为 'test_tool'，实际为: %s", def.Name)
	}

	if def.Description != "测试工具" {
		t.Errorf("期望定义描述为 '测试工具'，实际为: %s", def.Description)
	}

	if def.Parameters == nil {
		t.Error("定义参数不应该为空")
	}
}

func TestToolRegistry_UnregisterTool(t *testing.T) {
	registry := NewToolRegistry()
	tool := createMockTool("test_tool", "测试工具")

	// 先注册工具
	registry.RegisterTool(tool)

	if registry.Count() != 1 {
		t.Errorf("注册后期望工具数量为1，实际为: %d", registry.Count())
	}

	// 测试注销存在的工具
	err := registry.UnregisterTool("test_tool")
	if err != nil {
		t.Errorf("注销工具时发生错误: %v", err)
	}

	if registry.Count() != 0 {
		t.Errorf("注销后期望工具数量为0，实际为: %d", registry.Count())
	}

	// 测试注销不存在的工具
	err = registry.UnregisterTool("nonexistent_tool")
	if err == nil {
		t.Error("注销不存在的工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeToolNotFound) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeToolNotFound, util.GetErrorCode(err))
	}
}

func TestToolRegistry_UnregisterTool_EmptyName(t *testing.T) {
	registry := NewToolRegistry()

	err := registry.UnregisterTool("")
	if err == nil {
		t.Error("注销空名称工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeInvalidParam) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeInvalidParam, util.GetErrorCode(err))
	}
}

func TestToolRegistry_HasTool(t *testing.T) {
	registry := NewToolRegistry()
	tool := createMockTool("test_tool", "测试工具")

	// 测试不存在的工具
	if registry.HasTool("test_tool") {
		t.Error("不存在的工具应该返回false")
	}

	// 注册工具后测试
	registry.RegisterTool(tool)

	if !registry.HasTool("test_tool") {
		t.Error("存在的工具应该返回true")
	}

	if registry.HasTool("nonexistent_tool") {
		t.Error("不存在的工具应该返回false")
	}
}

func TestToolRegistry_Count(t *testing.T) {
	registry := NewToolRegistry()

	// 测试空注册表
	if registry.Count() != 0 {
		t.Errorf("空注册表计数应该为0，实际为: %d", registry.Count())
	}

	// 注册工具后测试
	tool1 := createMockTool("tool1", "工具1")
	tool2 := createMockTool("tool2", "工具2")

	registry.RegisterTool(tool1)
	if registry.Count() != 1 {
		t.Errorf("注册1个工具后计数应该为1，实际为: %d", registry.Count())
	}

	registry.RegisterTool(tool2)
	if registry.Count() != 2 {
		t.Errorf("注册2个工具后计数应该为2，实际为: %d", registry.Count())
	}

	registry.UnregisterTool("tool1")
	if registry.Count() != 1 {
		t.Errorf("注销1个工具后计数应该为1，实际为: %d", registry.Count())
	}
}
