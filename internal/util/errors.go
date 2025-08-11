package util

import (
	"ai-ops/internal/common/errors"
	"fmt"
)

// 错误代码常量 - 使用通用错误处理系统中的错误代码
// 为了向后兼容，保留这些常量，但它们指向通用错误处理系统中的错误代码
const (
	ErrCodeConfigNotFound       = errors.ErrCodeConfigNotFound       // 配置文件未找到
	ErrCodeConfigInvalid        = errors.ErrCodeConfigInvalid        // 配置文件无效
	ErrCodeConfigLoadFailed     = errors.ErrCodeConfigLoadFailed     // 配置加载失败
	ErrCodeConfigParseFailed    = errors.ErrCodeConfigParseFailed    // 配置解析失败
	ErrCodeModelNotFound        = errors.ErrCodeModelNotFound        // 模型未找到
	ErrCodeAPIKeyMissing        = errors.ErrCodeAPIKeyMissing        // API密钥缺失
	ErrCodeNetworkFailed        = errors.ErrCodeNetworkFailed        // 网络请求失败
	ErrCodeAPIRequestFailed     = errors.ErrCodeAPIRequestFailed     // API请求失败
	ErrCodeToolNotFound         = errors.ErrCodeToolNotFound         // 工具未找到
	ErrCodeToolExecutionFailed  = errors.ErrCodeToolExecutionFailed  // 工具执行失败
	ErrCodeInvalidParam         = errors.ErrCodeInvalidParam         // 无效参数
	ErrCodeInternalErr          = errors.ErrCodeInternalErr          // 内部错误
	ErrCodeAIResponseInvalid    = errors.ErrCodeAIResponseInvalid    // AI响应无效
	ErrCodeNotFound             = errors.ErrCodeNotFound             // 资源未找到
	ErrCodeInitializationFailed = errors.ErrCodeInitializationFailed // 初始化失败

	// MCP相关错误代码
	ErrCodeMCPNotConfigured    = errors.ErrCodeMCPNotConfigured    // MCP未配置
	ErrCodeMCPConnectionFailed = errors.ErrCodeMCPConnectionFailed // MCP连接失败
	ErrCodeMCPNotConnected     = errors.ErrCodeMCPNotConnected     // MCP未连接
	ErrCodeMCPToolListFailed   = errors.ErrCodeMCPToolListFailed   // MCP工具列表获取失败
	ErrCodeMCPToolCallFailed   = errors.ErrCodeMCPToolCallFailed   // MCP工具调用失败
)

// AppError 应用错误结构 - 使用通用错误处理系统中的AppError
type AppError = errors.AppError

// 创建新的应用错误 - 使用通用错误处理系统
func NewError(code, message string) *AppError {
	return errors.NewError(code, message)
}

// 创建带详情的应用错误 - 使用通用错误处理系统
func NewErrorWithDetail(code, message, details string) *AppError {
	return errors.NewErrorWithDetails(code, message, details)
}

// 包装现有错误 - 使用通用错误处理系统
func WrapError(code, message string, cause error) *AppError {
	return errors.WrapError(code, message, cause)
}

// 检查错误是否为指定类型 - 使用通用错误处理系统
func IsErrorCode(err error, code string) bool {
	return errors.IsErrorCode(err, code)
}

// 获取错误代码 - 使用通用错误处理系统
func GetErrorCode(err error) string {
	return errors.GetErrorCode(err)
}

// 获取用户友好的错误消息 - 使用通用错误处理系统
func GetUserFriendlyMessage(err error) string {
	return errors.GetUserFriendlyMessage(err)
}

// 错误恢复建议 - 已移除，使用 GetUserFriendlyMessage 替代
func GetRecoverySuggestion(err error) string {
	return errors.GetUserFriendlyMessage(err)
}

// 工具特定的错误创建函数

// NewToolError 创建工具错误
func NewToolError(message string) *AppError {
	return errors.NewToolError(message)
}

// NewToolErrorWithDetails 创建带详情的工具错误
func NewToolErrorWithDetails(message, details string) *AppError {
	return errors.NewToolErrorWithDetails(message, details)
}

// WrapToolError 包装现有错误为工具错误
func WrapToolError(message string, cause error) *AppError {
	return errors.WrapToolError(message, cause)
}

// NewToolNotFoundError 创建工具未找到错误
func NewToolNotFoundError(toolName string) *AppError {
	return errors.NewErrorWithDetails(errors.ErrCodeToolNotFound, "工具未找到",
		fmt.Sprintf("工具名称: %s", toolName))
}

// NewToolExecutionError 创建工具执行错误
func NewToolExecutionError(toolName string, cause error) *AppError {
	return errors.WrapErrorWithDetails(errors.ErrCodeToolExecutionFailed,
		fmt.Sprintf("工具 %s 执行失败", toolName), cause,
		fmt.Sprintf("工具名称: %s", toolName))
}
