package tools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"ai-ops/internal/util"
)

// ToolDiscovery 工具发现接口
type ToolDiscovery interface {
	// DiscoverTools 发现工具
	DiscoverTools() ([]Tool, error)

	// RegisterDiscoveredTools 注册发现的工具
	RegisterDiscoveredTools(manager ToolManager) error
}

// FunctionBasedTool 基于函数的工具实现
type FunctionBasedTool struct {
	name        string
	description string
	parameters  map[string]interface{}
	executeFunc func(ctx context.Context, args map[string]interface{}) (string, error)
}

func (f *FunctionBasedTool) Name() string {
	return f.name
}

func (f *FunctionBasedTool) Description() string {
	return f.description
}

func (f *FunctionBasedTool) Parameters() map[string]interface{} {
	return f.parameters
}

func (f *FunctionBasedTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return f.executeFunc(ctx, args)
}

// ToolDiscoveryRegistry 工具发现注册器
type ToolDiscoveryRegistry struct {
	tools []Tool
}

// NewToolDiscovery 创建新的工具发现器
func NewToolDiscovery() ToolDiscovery {
	return &ToolDiscoveryRegistry{
		tools: make([]Tool, 0),
	}
}

// RegisterFunction 注册函数为工具
func (r *ToolDiscoveryRegistry) RegisterFunction(name, description string, parameters map[string]interface{},
	executeFunc func(ctx context.Context, args map[string]interface{}) (string, error)) {

	tool := &FunctionBasedTool{
		name:        name,
		description: description,
		parameters:  parameters,
		executeFunc: executeFunc,
	}

	r.tools = append(r.tools, tool)
}

// DiscoverTools 发现已注册的工具
func (r *ToolDiscoveryRegistry) DiscoverTools() ([]Tool, error) {
	util.Infow("开始工具发现", map[string]interface{}{
		"registered_tools_count": len(r.tools),
	})

	discoveredTools := make([]Tool, len(r.tools))
	copy(discoveredTools, r.tools)

	util.Infow("工具发现完成", map[string]interface{}{
		"discovered_tools_count": len(discoveredTools),
	})

	return discoveredTools, nil
}

// RegisterDiscoveredTools 将发现的工具注册到管理器
func (r *ToolDiscoveryRegistry) RegisterDiscoveredTools(manager ToolManager) error {
	tools, err := r.DiscoverTools()
	if err != nil {
		return util.WrapError(util.ErrCodeInternalErr, "工具发现失败", err)
	}

	successCount := 0
	errorCount := 0

	for _, tool := range tools {
		err := manager.RegisterTool(tool)
		if err != nil {
			util.LogErrorWithFields(err, "工具注册失败", map[string]interface{}{
				"tool_name": tool.Name(),
			})
			errorCount++
			continue
		}
		successCount++
	}

	util.Infow("工具批量注册完成", map[string]interface{}{
		"success_count": successCount,
		"error_count":   errorCount,
		"total_count":   len(tools),
	})

	if errorCount > 0 {
		return util.NewErrorWithDetail(util.ErrCodeInternalErr, "部分工具注册失败",
			fmt.Sprintf("成功: %d, 失败: %d", successCount, errorCount))
	}

	return nil
}

// AutoDiscovery 自动发现器，用于自动扫描和注册工具
type AutoDiscovery struct {
	registries []ToolDiscovery
}

// NewAutoDiscovery 创建自动发现器
func NewAutoDiscovery() *AutoDiscovery {
	return &AutoDiscovery{
		registries: make([]ToolDiscovery, 0),
	}
}

// AddRegistry 添加工具注册器
func (a *AutoDiscovery) AddRegistry(registry ToolDiscovery) {
	a.registries = append(a.registries, registry)
}

// DiscoverAndRegisterAll 发现并注册所有工具
func (a *AutoDiscovery) DiscoverAndRegisterAll(manager ToolManager) error {
	totalTools := 0
	totalErrors := 0

	util.Infow("开始自动工具发现和注册", map[string]interface{}{
		"registry_count": len(a.registries),
	})

	for i, registry := range a.registries {
		util.Infow("处理工具注册器", map[string]interface{}{
			"registry_index":   i + 1,
			"total_registries": len(a.registries),
		})

		tools, err := registry.DiscoverTools()
		if err != nil {
			util.LogErrorWithFields(err, "工具发现失败", map[string]interface{}{
				"registry_index": i + 1,
			})
			totalErrors++
			continue
		}

		for _, tool := range tools {
			err := manager.RegisterTool(tool)
			if err != nil {
				util.LogErrorWithFields(err, "工具注册失败", map[string]interface{}{
					"tool_name":      tool.Name(),
					"registry_index": i + 1,
				})
				totalErrors++
				continue
			}
			totalTools++
		}
	}

	util.Infow("自动工具发现和注册完成", map[string]interface{}{
		"total_tools_registered": totalTools,
		"total_errors":           totalErrors,
	})

	if totalErrors > 0 {
		return util.NewErrorWithDetail(util.ErrCodeInternalErr, "部分工具注册失败",
			fmt.Sprintf("成功注册: %d, 错误: %d", totalTools, totalErrors))
	}

	return nil
}

// ToolExecutor 工具调用执行器
type ToolExecutor struct {
	manager ToolManager
}

// NewToolExecutor 创建工具执行器
func NewToolExecutor(manager ToolManager) *ToolExecutor {
	return &ToolExecutor{
		manager: manager,
	}
}

// ExecuteWithRetry 带重试的工具执行
func (e *ToolExecutor) ExecuteWithRetry(ctx context.Context, call ToolCall, maxRetries int) (string, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			util.Infow("重试工具执行", map[string]interface{}{
				"tool_name":   call.Name,
				"call_id":     call.ID,
				"attempt":     attempt,
				"max_retries": maxRetries,
			})
		}

		result, err := e.manager.ExecuteToolCall(ctx, call)
		if err == nil {
			if attempt > 0 {
				util.Infow("工具执行重试成功", map[string]interface{}{
					"tool_name":          call.Name,
					"call_id":            call.ID,
					"successful_attempt": attempt,
				})
			}
			return result, nil
		}

		lastErr = err

		// 检查是否应该重试
		if !shouldRetry(err) {
			util.Infow("工具执行错误不可重试", map[string]interface{}{
				"tool_name":  call.Name,
				"call_id":    call.ID,
				"error_code": util.GetErrorCode(err),
			})
			break
		}

		if attempt < maxRetries {
			util.LogErrorWithFields(err, "工具执行失败，准备重试", map[string]interface{}{
				"tool_name": call.Name,
				"call_id":   call.ID,
				"attempt":   attempt,
			})
		}
	}

	util.LogErrorWithFields(lastErr, "工具执行最终失败", map[string]interface{}{
		"tool_name":   call.Name,
		"call_id":     call.ID,
		"max_retries": maxRetries,
	})

	return "", util.WrapError(util.ErrCodeToolExecutionFailed,
		fmt.Sprintf("工具 %s 执行失败，已重试 %d 次", call.Name, maxRetries), lastErr)
}

// shouldRetry 判断错误是否应该重试
func shouldRetry(err error) bool {
	errorCode := util.GetErrorCode(err)

	// 网络错误和内部错误可以重试
	retryableCodes := []string{
		util.ErrCodeNetworkFailed,
		util.ErrCodeInternalErr,
	}

	for _, code := range retryableCodes {
		if errorCode == code {
			return true
		}
	}

	return false
}

// GetFunctionName 获取函数名称（用于调试）
func GetFunctionName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

// SimplifyFunctionName 简化函数名称
func SimplifyFunctionName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// ToolScanner 工具扫描器，用于扫描目录中的工具
type ToolScanner struct {
	scanPaths []string // 扫描路径列表
}

// NewToolScanner 创建工具扫描器
func NewToolScanner(scanPaths ...string) *ToolScanner {
	if len(scanPaths) == 0 {
		// 默认扫描当前目录
		scanPaths = []string{"."}
	}

	return &ToolScanner{
		scanPaths: scanPaths,
	}
}

// ScanForTools 扫描工具文件
func (s *ToolScanner) ScanForTools() ([]string, error) {
	var toolFiles []string

	for _, scanPath := range s.scanPaths {
		util.Infow("开始扫描工具目录", map[string]interface{}{
			"scan_path": scanPath,
		})

		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				util.LogErrorWithFields(err, "扫描文件时发生错误", map[string]interface{}{
					"file_path": path,
				})
				return nil // 继续扫描其他文件
			}

			// 跳过目录和非Go文件
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}

			// 跳过测试文件
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}

			// 检查文件是否包含工具函数
			if s.containsToolFunction(path) {
				toolFiles = append(toolFiles, path)
				util.Infow("发现工具文件", map[string]interface{}{
					"file_path": path,
				})
			}

			return nil
		})

		if err != nil {
			util.LogErrorWithFields(err, "扫描目录失败", map[string]interface{}{
				"scan_path": scanPath,
			})
			continue
		}
	}

	util.Infow("工具文件扫描完成", map[string]interface{}{
		"total_files": len(toolFiles),
		"files":       toolFiles,
	})

	return toolFiles, nil
}

// containsToolFunction 检查文件是否包含工具函数
func (s *ToolScanner) containsToolFunction(filePath string) bool {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		util.LogErrorWithFields(err, "解析Go文件失败", map[string]interface{}{
			"file_path": filePath,
		})
		return false
	}

	// 查找符合工具函数命名规范的函数
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			funcName := funcDecl.Name.Name

			// 检查函数名是否符合工具函数命名规范
			if s.isToolFunction(funcName, funcDecl) {
				return true
			}
		}
	}

	return false
}

// isToolFunction 判断函数是否为工具函数
func (s *ToolScanner) isToolFunction(funcName string, funcDecl *ast.FuncDecl) bool {
	// 工具函数命名规范：Call*Tool（至少9个字符，如CallXTool）或 *Tool（但不以Call开头）
	if strings.HasPrefix(funcName, "Call") && strings.HasSuffix(funcName, "Tool") && len(funcName) > 8 {
		return s.validateToolFunctionSignature(funcDecl)
	}

	if strings.HasSuffix(funcName, "Tool") && funcName != "Tool" && !strings.HasPrefix(funcName, "Call") {
		return s.validateToolFunctionSignature(funcDecl)
	}

	return false
}

// validateToolFunctionSignature 验证工具函数签名
func (s *ToolScanner) validateToolFunctionSignature(funcDecl *ast.FuncDecl) bool {
	// 检查函数类型
	if funcDecl.Type == nil || funcDecl.Type.Params == nil || funcDecl.Type.Results == nil {
		return false
	}

	// 检查参数：应该有context.Context和map[string]any
	params := funcDecl.Type.Params.List
	if len(params) != 2 {
		return false
	}

	// 检查返回值：应该返回(string, error)
	results := funcDecl.Type.Results.List
	if len(results) != 2 {
		return false
	}

	return true
}

// AutoToolRegistry 自动工具注册器
type AutoToolRegistry struct {
	scanner   *ToolScanner
	discovery *ToolDiscoveryRegistry
}

// NewAutoToolRegistry 创建自动工具注册器
func NewAutoToolRegistry(scanPaths ...string) *AutoToolRegistry {
	return &AutoToolRegistry{
		scanner:   NewToolScanner(scanPaths...),
		discovery: NewToolDiscovery().(*ToolDiscoveryRegistry),
	}
}

// AutoRegisterTools 自动注册工具
func (r *AutoToolRegistry) AutoRegisterTools(manager ToolManager) error {
	util.Infow("开始自动工具注册", map[string]interface{}{
		"scan_paths": r.scanner.scanPaths,
	})

	// 扫描工具文件
	toolFiles, err := r.scanner.ScanForTools()
	if err != nil {
		return util.WrapError(util.ErrCodeInternalErr, "工具文件扫描失败", err)
	}

	if len(toolFiles) == 0 {
		util.Infow("未发现工具文件", map[string]interface{}{})
		return nil
	}

	// 注册已知的工具函数
	registeredCount := 0

	// 这里可以根据扫描结果动态注册工具
	// 目前先注册已知的天气工具作为示例
	if r.containsWeatherTool(toolFiles) {
		r.registerWeatherTool()
		registeredCount++
	}

	// 将发现的工具注册到管理器
	err = r.discovery.RegisterDiscoveredTools(manager)
	if err != nil {
		return util.WrapError(util.ErrCodeInternalErr, "工具注册失败", err)
	}

	util.Infow("自动工具注册完成", map[string]interface{}{
		"registered_count": registeredCount,
		"total_files":      len(toolFiles),
	})

	return nil
}

// containsWeatherTool 检查是否包含天气工具
func (r *AutoToolRegistry) containsWeatherTool(toolFiles []string) bool {
	for _, file := range toolFiles {
		if strings.Contains(file, "weather.go") {
			return true
		}
	}
	return false
}

// registerWeatherTool 注册天气工具
func (r *AutoToolRegistry) registerWeatherTool() {
	// 创建天气工具实例
	weatherTool := NewWeatherTool()

	// 将天气工具添加到发现列表中
	r.discovery.tools = append(r.discovery.tools, weatherTool)

	util.Infow("天气工具已注册到自动发现器", map[string]interface{}{
		"tool_name":   weatherTool.Name(),
		"description": weatherTool.Description(),
	})
}

// ToolCallExecutor 工具调用执行器（增强版）
type ToolCallExecutor struct {
	manager    ToolManager
	maxRetries int
	retryDelay int // 重试延迟（毫秒）
}

// NewToolCallExecutor 创建工具调用执行器
func NewToolCallExecutor(manager ToolManager) *ToolCallExecutor {
	return &ToolCallExecutor{
		manager:    manager,
		maxRetries: 3,
		retryDelay: 1000, // 1秒
	}
}

// SetRetryConfig 设置重试配置
func (e *ToolCallExecutor) SetRetryConfig(maxRetries, retryDelayMs int) {
	e.maxRetries = maxRetries
	e.retryDelay = retryDelayMs
}

// ExecuteWithTimeout 带超时的工具执行
func (e *ToolCallExecutor) ExecuteWithTimeout(ctx context.Context, call ToolCall, timeoutMs int) (string, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	util.Infow("开始带超时的工具执行", map[string]interface{}{
		"tool_name":  call.Name,
		"call_id":    call.ID,
		"timeout_ms": timeoutMs,
	})

	result, err := e.manager.ExecuteToolCall(timeoutCtx, call)

	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			util.LogErrorWithFields(err, "工具执行超时", map[string]interface{}{
				"tool_name":  call.Name,
				"call_id":    call.ID,
				"timeout_ms": timeoutMs,
			})
			return "", util.NewErrorWithDetail(util.ErrCodeToolExecutionFailed, "工具执行超时",
				fmt.Sprintf("工具 %s 执行超过 %d 毫秒", call.Name, timeoutMs))
		}
		return "", err
	}

	util.Infow("带超时的工具执行成功", map[string]interface{}{
		"tool_name":     call.Name,
		"call_id":       call.ID,
		"result_length": len(result),
	})

	return result, nil
}
