package tools

import (
	"context"
	"errors"
	"testing"
	"time"

	"ai-ops/internal/util"
)

func TestNewToolManager(t *testing.T) {
	manager := NewToolManager()

	if manager == nil {
		t.Error("工具管理器不应该为空")
	}

	// 测试初始状态
	tools := manager.GetTools()
	if len(tools) != 0 {
		t.Errorf("新管理器应该没有工具，实际数量: %d", len(tools))
	}
}

func TestDefaultToolManager_RegisterTool(t *testing.T) {
	manager := NewToolManager()
	tool := createMockTool("test_tool", "测试工具")

	err := manager.RegisterTool(tool)
	if err != nil {
		t.Errorf("注册工具时发生错误: %v", err)
	}

	tools := manager.GetTools()
	if len(tools) != 1 {
		t.Errorf("期望工具数量为1，实际为: %d", len(tools))
	}

	if tools[0].Name() != "test_tool" {
		t.Errorf("期望工具名称为 'test_tool'，实际为: %s", tools[0].Name())
	}
}

func TestDefaultToolManager_GetTool(t *testing.T) {
	manager := NewToolManager()
	originalTool := createMockTool("test_tool", "测试工具")

	// 先注册工具
	manager.RegisterTool(originalTool)

	// 测试获取存在的工具
	tool, err := manager.GetTool("test_tool")
	if err != nil {
		t.Errorf("获取工具时发生错误: %v", err)
	}

	if tool.Name() != "test_tool" {
		t.Errorf("期望工具名称为 'test_tool'，实际为: %s", tool.Name())
	}

	// 测试获取不存在的工具
	_, err = manager.GetTool("nonexistent_tool")
	if err == nil {
		t.Error("获取不存在的工具应该返回错误")
	}
}

func TestDefaultToolManager_GetToolDefinitions(t *testing.T) {
	manager := NewToolManager()
	tool := createMockTool("test_tool", "测试工具")

	// 注册工具
	manager.RegisterTool(tool)

	definitions := manager.GetToolDefinitions()
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
}

func TestDefaultToolManager_ExecuteToolCall_Success(t *testing.T) {
	manager := NewToolManager()

	// 创建自定义执行函数的工具
	tool := &MockTool{
		name:        "echo_tool",
		description: "回显工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			input, ok := args["message"].(string)
			if !ok {
				return "", errors.New("缺少message参数")
			}
			return "echo: " + input, nil
		},
	}

	// 注册工具
	manager.RegisterTool(tool)

	// 创建工具调用
	call := ToolCall{
		ID:   "call_123",
		Name: "echo_tool",
		Arguments: map[string]interface{}{
			"message": "hello world",
		},
	}

	// 执行工具调用
	ctx := context.Background()
	result, err := manager.ExecuteToolCall(ctx, call)

	if err != nil {
		t.Errorf("执行工具调用时发生错误: %v", err)
	}

	expected := "echo: hello world"
	if result != expected {
		t.Errorf("期望结果为 '%s'，实际为: '%s'", expected, result)
	}
}

func TestDefaultToolManager_ExecuteToolCall_ToolNotFound(t *testing.T) {
	manager := NewToolManager()

	// 创建调用不存在工具的请求
	call := ToolCall{
		ID:   "call_123",
		Name: "nonexistent_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	_, err := manager.ExecuteToolCall(ctx, call)

	if err == nil {
		t.Error("调用不存在的工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeToolNotFound) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeToolNotFound, util.GetErrorCode(err))
	}
}

func TestDefaultToolManager_ExecuteToolCall_ExecutionError(t *testing.T) {
	manager := NewToolManager()

	// 创建会返回错误的工具
	tool := &MockTool{
		name:        "error_tool",
		description: "错误工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "", errors.New("执行失败")
		},
	}

	// 注册工具
	manager.RegisterTool(tool)

	// 创建工具调用
	call := ToolCall{
		ID:   "call_123",
		Name: "error_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	_, err := manager.ExecuteToolCall(ctx, call)

	if err == nil {
		t.Error("工具执行失败应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeToolExecutionFailed) {
		t.Errorf("期望错误代码为 %s，实际为: %s", util.ErrCodeToolExecutionFailed, util.GetErrorCode(err))
	}
}

func TestDefaultToolManager_ExecuteToolCallWithResult_Success(t *testing.T) {
	manager := NewToolManager().(*DefaultToolManager)

	// 创建简单的工具
	tool := createMockTool("simple_tool", "简单工具")
	manager.RegisterTool(tool)

	// 创建工具调用
	call := ToolCall{
		ID:   "call_123",
		Name: "simple_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	result := manager.ExecuteToolCallWithResult(ctx, call)

	if result == nil {
		t.Error("结果不应该为空")
		return
	}

	if !result.Success {
		t.Errorf("期望执行成功，实际失败: %s", result.Error)
	}

	if result.Result != "mock result" {
		t.Errorf("期望结果为 'mock result'，实际为: '%s'", result.Result)
	}

	if result.ExecutionTime <= 0 {
		t.Error("执行时间应该大于0")
	}
}

func TestDefaultToolManager_ExecuteToolCallWithResult_Failure(t *testing.T) {
	manager := NewToolManager().(*DefaultToolManager)

	// 创建会返回错误的工具
	tool := &MockTool{
		name:        "error_tool",
		description: "错误工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "", errors.New("执行失败")
		},
	}

	manager.RegisterTool(tool)

	// 创建工具调用
	call := ToolCall{
		ID:   "call_123",
		Name: "error_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	result := manager.ExecuteToolCallWithResult(ctx, call)

	if result == nil {
		t.Error("结果不应该为空")
		return
	}

	if result.Success {
		t.Error("期望执行失败，实际成功")
	}

	if result.Error == "" {
		t.Error("失败结果应该包含错误信息")
	}

	if result.Result != "" {
		t.Errorf("失败结果不应该有结果内容，实际为: '%s'", result.Result)
	}

	if result.ExecutionTime <= 0 {
		t.Error("执行时间应该大于0")
	}
}

func TestDefaultToolManager_ExecuteToolCall_WithContext(t *testing.T) {
	manager := NewToolManager()

	// 创建会检查上下文的工具
	tool := &MockTool{
		name:        "context_tool",
		description: "上下文工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			// 检查上下文是否被传递
			if ctx == nil {
				return "", errors.New("上下文为空")
			}

			// 模拟一些处理时间
			select {
			case <-time.After(time.Millisecond * 10):
				return "context processed", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
	}

	manager.RegisterTool(tool)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	call := ToolCall{
		ID:   "call_123",
		Name: "context_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	result, err := manager.ExecuteToolCall(ctx, call)

	if err != nil {
		t.Errorf("执行工具调用时发生错误: %v", err)
	}

	if result != "context processed" {
		t.Errorf("期望结果为 'context processed'，实际为: '%s'", result)
	}
}

func TestDefaultToolManager_ExecuteToolCall_ContextCancellation(t *testing.T) {
	manager := NewToolManager()

	// 创建长时间运行的工具
	tool := &MockTool{
		name:        "long_running_tool",
		description: "长时间运行工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			select {
			case <-time.After(time.Second * 2): // 模拟长时间处理
				return "completed", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
	}

	manager.RegisterTool(tool)

	// 创建会很快取消的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	call := ToolCall{
		ID:   "call_123",
		Name: "long_running_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	_, err := manager.ExecuteToolCall(ctx, call)

	if err == nil {
		t.Error("上下文取消应该导致错误")
	}

	// 检查是否是上下文取消错误
	if !errors.Is(err, context.DeadlineExceeded) && !util.IsErrorCode(err, util.ErrCodeToolExecutionFailed) {
		t.Errorf("期望上下文取消错误，实际错误: %v", err)
	}
}
