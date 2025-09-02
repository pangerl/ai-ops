package tools

import (
	"context"
	"fmt"
	"slices"
	"time"

	"ai-ops/internal/util"
	"ai-ops/internal/util/errors"
)

// ToolCallExecutor 工具调用执行器（增强版）
// 它包装了 ToolManager，为工具执行增加了超时和重试等高级功能。
type ToolCallExecutor struct {
	manager    ToolManager
	maxRetries int
	retryDelay time.Duration // 重试延迟
}

// SetRetryConfig 设置重试配置
func (e *ToolCallExecutor) SetRetryConfig(maxRetries int, retryDelayMs int) {
	e.maxRetries = maxRetries
	e.retryDelay = time.Duration(retryDelayMs) * time.Millisecond
}

// ExecuteWithRetryAndTimeout 使用重试和超时机制执行工具调用
func (e *ToolCallExecutor) ExecuteWithRetryAndTimeout(ctx context.Context, call ToolCall, timeoutMs int) (string, error) {
	var result string
	var err error

	for i := 0; i <= e.maxRetries; i++ {
		// 创建带超时的上下文
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)

		util.Debugw("开始工具执行（尝试）", map[string]any{
			"tool_name":  call.Name,
			"call_id":    call.ID,
			"attempt":    i + 1,
			"max_tries":  e.maxRetries + 1,
			"timeout_ms": timeoutMs,
		})

		result, err = e.manager.ExecuteToolCall(timeoutCtx, call)

		// 检查错误
		if err != nil {
			// 检查是否是超时错误
			if timeoutCtx.Err() == context.DeadlineExceeded {
				// 先释放本轮上下文资源
				cancel()
				util.LogErrorWithFields(err, "工具执行超时", map[string]any{
					"tool_name":  call.Name,
					"call_id":    call.ID,
					"timeout_ms": timeoutMs,
				})
				// 超时是最终错误，不应重试
				return "", errors.NewToolErrorWithDetails("工具执行超时",
					fmt.Sprintf("工具 %s 执行超过 %d 毫秒", call.Name, timeoutMs))
			}

			// 判断是否是可重试的错误
			if i < e.maxRetries && shouldRetry(err) {
				// 释放本轮上下文资源再重试
				cancel()
				util.Warnw("工具执行失败，准备重试", map[string]any{
					"tool_name":   call.Name,
					"call_id":     call.ID,
					"error":       err.Error(),
					"retry_delay": e.retryDelay,
				})
				time.Sleep(e.retryDelay)
				continue // 进入下一次重试
			}

			// 不可重试或已达最大次数
			cancel()
			return "", err
		}

		// 执行成功，释放本轮上下文资源并返回结果
		cancel()
		util.Debugw("工具执行成功", map[string]any{
			"tool_name":     call.Name,
			"call_id":       call.ID,
			"result_length": len(result),
		})
		return result, nil
	}

	// 正常情况下不应到达这里，但为了代码健壮性，返回最后一次的错误
	return "", err
}

// shouldRetry 判断错误是否应该重试
func shouldRetry(err error) bool {
	errorCode := errors.GetErrorCode(err)

	// 定义可重试的错误码
	retryableCodes := []string{
		errors.ErrCodeNetworkFailed,
		errors.ErrCodeInternalErr,
	}

	return slices.Contains(retryableCodes, errorCode)
}
