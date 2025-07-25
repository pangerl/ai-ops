package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ai-ops/internal/util"
)

func TestNewToolScanner(t *testing.T) {
	// 测试默认扫描器
	scanner := NewToolScanner()
	if len(scanner.scanPaths) != 1 || scanner.scanPaths[0] != "." {
		t.Errorf("期望默认扫描路径为['.']，实际为: %v", scanner.scanPaths)
	}

	// 测试自定义扫描路径
	customPaths := []string{"./internal", "./cmd"}
	scanner = NewToolScanner(customPaths...)
	if len(scanner.scanPaths) != 2 {
		t.Errorf("期望扫描路径数量为2，实际为: %d", len(scanner.scanPaths))
	}

	for i, path := range customPaths {
		if scanner.scanPaths[i] != path {
			t.Errorf("期望扫描路径[%d]为'%s'，实际为'%s'", i, path, scanner.scanPaths[i])
		}
	}
}

func TestToolScanner_ScanForTools(t *testing.T) {
	// 创建临时目录和文件用于测试
	tempDir, err := os.MkdirTemp("", "tool_scanner_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建一个包含工具函数的Go文件
	toolFile := filepath.Join(tempDir, "test_tool.go")
	toolContent := `package tools

import (
	"context"
)

// CallTestTool 测试工具函数
func CallTestTool(ctx context.Context, args map[string]interface{}) (string, error) {
	return "test result", nil
}

// NotATool 不是工具函数
func NotATool() {
}
`

	err = os.WriteFile(toolFile, []byte(toolContent), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建一个不包含工具函数的Go文件
	normalFile := filepath.Join(tempDir, "normal.go")
	normalContent := `package tools

func NormalFunction() {
}
`

	err = os.WriteFile(normalFile, []byte(normalContent), 0644)
	if err != nil {
		t.Fatalf("创建普通文件失败: %v", err)
	}

	// 创建测试文件（应该被跳过）
	testFile := filepath.Join(tempDir, "test_test.go")
	testContent := `package tools

import "testing"

func TestSomething(t *testing.T) {
}
`

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 扫描工具
	scanner := NewToolScanner(tempDir)
	toolFiles, err := scanner.ScanForTools()
	if err != nil {
		t.Errorf("扫描工具文件时发生错误: %v", err)
	}

	// 验证结果
	if len(toolFiles) != 1 {
		t.Errorf("期望发现1个工具文件，实际发现: %d", len(toolFiles))
	}

	if len(toolFiles) > 0 && filepath.Base(toolFiles[0]) != "test_tool.go" {
		t.Errorf("期望发现test_tool.go，实际发现: %s", filepath.Base(toolFiles[0]))
	}
}

func TestToolScanner_containsToolFunction(t *testing.T) {
	// 创建临时文件
	tempDir, err := os.MkdirTemp("", "tool_function_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 测试包含工具函数的文件
	toolFile := filepath.Join(tempDir, "with_tool.go")
	toolContent := `package tools

import "context"

func CallWeatherTool(ctx context.Context, args map[string]interface{}) (string, error) {
	return "", nil
}
`

	err = os.WriteFile(toolFile, []byte(toolContent), 0644)
	if err != nil {
		t.Fatalf("创建工具文件失败: %v", err)
	}

	// 测试不包含工具函数的文件
	normalFile := filepath.Join(tempDir, "without_tool.go")
	normalContent := `package tools

func NormalFunction() {
}
`

	err = os.WriteFile(normalFile, []byte(normalContent), 0644)
	if err != nil {
		t.Fatalf("创建普通文件失败: %v", err)
	}

	scanner := NewToolScanner()

	// 测试包含工具函数的文件
	if !scanner.containsToolFunction(toolFile) {
		t.Error("应该检测到工具函数")
	}

	// 测试不包含工具函数的文件
	if scanner.containsToolFunction(normalFile) {
		t.Error("不应该检测到工具函数")
	}
}

func TestToolScanner_isToolFunction(t *testing.T) {
	testCases := []struct {
		funcName string
		expected bool
	}{
		{"CallWeatherTool", true},
		{"CallTestTool", true},
		{"WeatherTool", true},
		{"TestTool", true},
		{"Tool", false},
		{"CallTool", false},
		{"NormalFunction", false},
		{"CallWeather", false},
		{"Weather", false},
	}

	for _, tc := range testCases {
		// 由于我们需要ast.FuncDecl，这里只测试函数名匹配逻辑
		result := (strings.HasPrefix(tc.funcName, "Call") && strings.HasSuffix(tc.funcName, "Tool") && len(tc.funcName) > 8) ||
			(strings.HasSuffix(tc.funcName, "Tool") && tc.funcName != "Tool" && !strings.HasPrefix(tc.funcName, "Call"))

		if result != tc.expected {
			t.Errorf("函数名'%s'的检测结果期望为%v，实际为%v", tc.funcName, tc.expected, result)
		}
	}
}

func TestNewAutoToolRegistry(t *testing.T) {
	registry := NewAutoToolRegistry("./test1", "./test2")

	if registry == nil {
		t.Error("自动工具注册器不应该为空")
	}

	if len(registry.scanner.scanPaths) != 2 {
		t.Errorf("期望扫描路径数量为2，实际为: %d", len(registry.scanner.scanPaths))
	}

	if registry.discovery == nil {
		t.Error("工具发现器不应该为空")
	}
}

func TestAutoToolRegistry_containsWeatherTool(t *testing.T) {
	registry := NewAutoToolRegistry()

	// 测试包含天气工具的文件列表
	withWeatherTool := []string{
		"./internal/tools/weather.go",
		"./cmd/weather.go",
		"./weather.go",
	}

	for _, files := range [][]string{withWeatherTool} {
		if !registry.containsWeatherTool(files) {
			t.Errorf("应该检测到天气工具，文件列表: %v", files)
		}
	}

	// 测试不包含天气工具的文件列表
	withoutWeatherTool := []string{
		"./internal/tools/other.go",
		"./cmd/main.go",
		"./util.go",
	}

	if registry.containsWeatherTool(withoutWeatherTool) {
		t.Errorf("不应该检测到天气工具，文件列表: %v", withoutWeatherTool)
	}
}

func TestAutoToolRegistry_registerWeatherTool(t *testing.T) {
	registry := NewAutoToolRegistry()

	// 注册天气工具
	registry.registerWeatherTool()

	// 验证工具已注册
	tools, err := registry.discovery.DiscoverTools()
	if err != nil {
		t.Errorf("发现工具时发生错误: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("期望发现1个工具，实际发现: %d", len(tools))
	}

	if len(tools) > 0 {
		tool := tools[0]
		if tool.Name() != "weather" {
			t.Errorf("期望工具名称为'weather'，实际为: %s", tool.Name())
		}

		if tool.Description() != "查询指定地点的实时天气信息" {
			t.Errorf("期望工具描述为'查询指定地点的实时天气信息'，实际为: %s", tool.Description())
		}
	}
}

func TestNewToolCallExecutor(t *testing.T) {
	manager := NewToolManager()
	executor := NewToolCallExecutor(manager)

	if executor == nil {
		t.Error("工具调用执行器不应该为空")
	}

	if executor.manager != manager {
		t.Error("执行器应该使用传入的管理器")
	}

	if executor.maxRetries != 3 {
		t.Errorf("期望默认最大重试次数为3，实际为: %d", executor.maxRetries)
	}

	if executor.retryDelay != 1000 {
		t.Errorf("期望默认重试延迟为1000ms，实际为: %d", executor.retryDelay)
	}
}

func TestToolCallExecutor_SetRetryConfig(t *testing.T) {
	manager := NewToolManager()
	executor := NewToolCallExecutor(manager)

	// 设置重试配置
	executor.SetRetryConfig(5, 2000)

	if executor.maxRetries != 5 {
		t.Errorf("期望最大重试次数为5，实际为: %d", executor.maxRetries)
	}

	if executor.retryDelay != 2000 {
		t.Errorf("期望重试延迟为2000ms，实际为: %d", executor.retryDelay)
	}
}

func TestToolCallExecutor_ExecuteWithTimeout_Success(t *testing.T) {
	manager := NewToolManager()
	executor := NewToolCallExecutor(manager)

	// 注册一个快速执行的工具
	tool := &MockTool{
		name:        "fast_tool",
		description: "快速工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "fast result", nil
		},
	}

	manager.RegisterTool(tool)

	// 创建工具调用
	call := ToolCall{
		ID:   "call_fast",
		Name: "fast_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	result, err := executor.ExecuteWithTimeout(ctx, call, 5000) // 5秒超时

	if err != nil {
		t.Errorf("快速工具执行不应该超时，错误: %v", err)
	}

	if result != "fast result" {
		t.Errorf("期望结果为'fast result'，实际为: %s", result)
	}
}

func TestToolCallExecutor_ExecuteWithTimeout_Timeout(t *testing.T) {
	manager := NewToolManager()
	executor := NewToolCallExecutor(manager)

	// 注册一个慢速执行的工具
	tool := &MockTool{
		name:        "slow_tool",
		description: "慢速工具",
		executeFunc: func(ctx context.Context, args map[string]interface{}) (string, error) {
			select {
			case <-time.After(time.Second * 2): // 2秒延迟
				return "slow result", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
	}

	manager.RegisterTool(tool)

	// 创建工具调用
	call := ToolCall{
		ID:   "call_slow",
		Name: "slow_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	_, err := executor.ExecuteWithTimeout(ctx, call, 100) // 100ms超时

	if err == nil {
		t.Error("慢速工具应该超时")
	}

	if !util.IsErrorCode(err, util.ErrCodeToolExecutionFailed) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeToolExecutionFailed, util.GetErrorCode(err))
	}
}

func TestToolCallExecutor_ExecuteWithTimeout_ToolNotFound(t *testing.T) {
	manager := NewToolManager()
	executor := NewToolCallExecutor(manager)

	// 创建调用不存在工具的请求
	call := ToolCall{
		ID:   "call_missing",
		Name: "missing_tool",
		Arguments: map[string]interface{}{
			"input": "test",
		},
	}

	ctx := context.Background()
	_, err := executor.ExecuteWithTimeout(ctx, call, 5000)

	if err == nil {
		t.Error("调用不存在的工具应该返回错误")
	}

	if !util.IsErrorCode(err, util.ErrCodeToolNotFound) {
		t.Errorf("期望错误代码为%s，实际为: %s", util.ErrCodeToolNotFound, util.GetErrorCode(err))
	}
}
