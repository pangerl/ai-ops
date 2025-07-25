package tools

import (
	"context"
	"testing"
	"time"
)

// MockTool 模拟工具实现，用于测试
type MockTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	executeFunc func(ctx context.Context, args map[string]interface{}) (string, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Parameters() map[string]interface{} {
	return m.parameters
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return "mock result", nil
}

// 创建测试用的模拟工具
func createMockTool(name, description string) *MockTool {
	return &MockTool{
		name:        name,
		description: description,
		parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "输入参数",
				},
			},
			"required": []string{"input"},
		},
	}
}

func TestMockTool_Name(t *testing.T) {
	tool := createMockTool("test_tool", "测试工具")

	if tool.Name() != "test_tool" {
		t.Errorf("期望工具名称为 'test_tool'，实际为 '%s'", tool.Name())
	}
}

func TestMockTool_Description(t *testing.T) {
	tool := createMockTool("test_tool", "测试工具")

	if tool.Description() != "测试工具" {
		t.Errorf("期望工具描述为 '测试工具'，实际为 '%s'", tool.Description())
	}
}

func TestMockTool_Parameters(t *testing.T) {
	tool := createMockTool("test_tool", "测试工具")
	params := tool.Parameters()

	if params == nil {
		t.Error("参数不应该为空")
		return
	}

	if params["type"] != "object" {
		t.Errorf("期望参数类型为 'object'，实际为 '%v'", params["type"])
	}
}

func TestMockTool_Execute(t *testing.T) {
	tool := createMockTool("test_tool", "测试工具")
	ctx := context.Background()
	args := map[string]interface{}{
		"input": "test input",
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Errorf("执行工具时发生错误: %v", err)
	}

	if result != "mock result" {
		t.Errorf("期望结果为 'mock result'，实际为 '%s'", result)
	}
}

func TestMockTool_ExecuteWithCustomFunc(t *testing.T) {
	tool := &MockTool{
		name:        "custom_tool",
		description: "自定义工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			input, ok := args["input"].(string)
			if !ok {
				return "", nil
			}
			return "processed: " + input, nil
		},
	}

	ctx := context.Background()
	args := map[string]interface{}{
		"input": "test data",
	}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Errorf("执行工具时发生错误: %v", err)
	}

	expected := "processed: test data"
	if result != expected {
		t.Errorf("期望结果为 '%s'，实际为 '%s'", expected, result)
	}
}

func TestToolDefinition(t *testing.T) {
	def := ToolDefinition{
		Name:        "test_tool",
		Description: "测试工具",
		Parameters: map[string]interface{}{
			"type": "object",
		},
	}

	if def.Name != "test_tool" {
		t.Errorf("期望工具定义名称为 'test_tool'，实际为 '%s'", def.Name)
	}

	if def.Description != "测试工具" {
		t.Errorf("期望工具定义描述为 '测试工具'，实际为 '%s'", def.Description)
	}

	if def.Parameters["type"] != "object" {
		t.Errorf("期望参数类型为 'object'，实际为 '%v'", def.Parameters["type"])
	}
}

func TestToolResult(t *testing.T) {
	// 测试成功结果
	successResult := ToolResult{
		Success:       true,
		Result:        "success result",
		ExecutionTime: time.Millisecond * 100,
	}

	if !successResult.Success {
		t.Error("期望结果为成功")
	}

	if successResult.Result != "success result" {
		t.Errorf("期望结果为 'success result'，实际为 '%s'", successResult.Result)
	}

	if successResult.Error != "" {
		t.Errorf("成功结果不应该有错误信息，实际为 '%s'", successResult.Error)
	}

	// 测试失败结果
	failureResult := ToolResult{
		Success:       false,
		Error:         "execution failed",
		ExecutionTime: time.Millisecond * 50,
	}

	if failureResult.Success {
		t.Error("期望结果为失败")
	}

	if failureResult.Error != "execution failed" {
		t.Errorf("期望错误信息为 'execution failed'，实际为 '%s'", failureResult.Error)
	}

	if failureResult.Result != "" {
		t.Errorf("失败结果不应该有结果内容，实际为 '%s'", failureResult.Result)
	}
}

func TestToolCall(t *testing.T) {
	call := ToolCall{
		ID:   "call_123",
		Name: "test_tool",
		Arguments: map[string]interface{}{
			"input": "test input",
		},
	}

	if call.ID != "call_123" {
		t.Errorf("期望调用ID为 'call_123'，实际为 '%s'", call.ID)
	}

	if call.Name != "test_tool" {
		t.Errorf("期望工具名称为 'test_tool'，实际为 '%s'", call.Name)
	}

	if call.Arguments["input"] != "test input" {
		t.Errorf("期望参数input为 'test input'，实际为 '%v'", call.Arguments["input"])
	}
}
