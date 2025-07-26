package tools

import (
	"context"
	"fmt"
	"slices"
	"time"

	"ai-ops/internal/util"
)

// FunctionBasedTool 基于函数的工具实现
type FunctionBasedTool struct {
	name        string
	description string
	parameters  map[string]any
	executeFunc func(ctx context.Context, args map[string]any) (string, error)
}

func (f *FunctionBasedTool) Name() string {
	return f.name
}

func (f *FunctionBasedTool) Description() string {
	return f.description
}

func (f *FunctionBasedTool) Parameters() map[string]any {
	return f.parameters
}

func (f *FunctionBasedTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	return f.executeFunc(ctx, args)
}

// ToolDiscoveryRegistry 工具发现注册器（用于手动注册基于函数的工具）
type ToolDiscoveryRegistry struct {
	tools []Tool
}

// NewToolDiscovery 创建新的工具发现器
func NewToolDiscovery() *ToolDiscoveryRegistry {
	return &ToolDiscoveryRegistry{
		tools: make([]Tool, 0),
	}
}

// RegisterFunction 注册函数为工具
func (r *ToolDiscoveryRegistry) RegisterFunction(name, description string, parameters map[string]any,
	executeFunc func(ctx context.Context, args map[string]any) (string, error)) {

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
	util.Infow("开始工具发现", map[string]any{
		"registered_tools_count": len(r.tools),
	})

	discoveredTools := make([]Tool, len(r.tools))
	copy(discoveredTools, r.tools)

	util.Infow("工具发现完成", map[string]any{
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
			util.LogErrorWithFields(err, "工具注册失败", map[string]any{
				"tool_name": tool.Name(),
			})
			errorCount++
			continue
		}
		successCount++
	}

	util.Infow("工具批量注册完成", map[string]any{
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

// 删除了ToolExecutor，使用ToolManager直接执行即可

// shouldRetry 判断错误是否应该重试
func shouldRetry(err error) bool {
	errorCode := util.GetErrorCode(err)

	// 网络错误和内部错误可以重试
	retryableCodes := []string{
		util.ErrCodeNetworkFailed,
		util.ErrCodeInternalErr,
	}

	return slices.Contains(retryableCodes, errorCode)

	return false
}

// AutoToolRegistry 自动工具注册器（简化版）
type AutoToolRegistry struct {
	pluginLoader *SimplePluginLoader
}

// NewAutoToolRegistry 创建自动工具注册器
func NewAutoToolRegistry(pluginDir string) *AutoToolRegistry {
	return &AutoToolRegistry{
		pluginLoader: NewSimplePluginLoader(pluginDir),
	}
}

// AutoRegisterTools 自动注册工具
func (r *AutoToolRegistry) AutoRegisterTools(manager ToolManager) error {
	util.Infow("开始自动工具注册", map[string]any{
		"plugin_dir": r.pluginLoader.pluginDir,
	})

	// 使用插件加载器加载工具
	err := r.pluginLoader.LoadPlugins(manager)
	if err != nil {
		return util.WrapError(util.ErrCodeInternalErr, "插件加载失败", err)
	}

	util.Infow("自动工具注册完成", map[string]any{})
	return nil
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

	util.Infow("开始带超时的工具执行", map[string]any{
		"tool_name":  call.Name,
		"call_id":    call.ID,
		"timeout_ms": timeoutMs,
	})

	result, err := e.manager.ExecuteToolCall(timeoutCtx, call)

	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			util.LogErrorWithFields(err, "工具执行超时", map[string]any{
				"tool_name":  call.Name,
				"call_id":    call.ID,
				"timeout_ms": timeoutMs,
			})
			return "", util.NewErrorWithDetail(util.ErrCodeToolExecutionFailed, "工具执行超时",
				fmt.Sprintf("工具 %s 执行超过 %d 毫秒", call.Name, timeoutMs))
		}
		return "", err
	}

	util.Infow("带超时的工具执行成功", map[string]any{
		"tool_name":     call.Name,
		"call_id":       call.ID,
		"result_length": len(result),
	})

	return result, nil
}
