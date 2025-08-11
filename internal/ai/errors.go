package ai

import (
	"ai-ops/internal/common/errors"
	stderrors "errors"
)

// 为了保持向后兼容性，保留原有的错误变量定义，但使用通用错误系统
var (
	ErrClientNotFound    = stderrors.New("AI client not found")
	ErrInvalidConfig     = stderrors.New("invalid AI client configuration")
	ErrAPIKeyMissing     = stderrors.New("API key is missing")
	ErrNetworkFailed     = stderrors.New("network request failed")
	ErrInvalidResponse   = stderrors.New("invalid response from AI service")
	ErrToolCallFailed    = stderrors.New("tool call execution failed")
	ErrContextCanceled   = stderrors.New("request context was canceled")
	ErrTimeout           = stderrors.New("request timeout")
	ErrRateLimited       = stderrors.New("rate limit exceeded")
	ErrModelNotSupported = stderrors.New("model not supported")
	ErrInvalidParameters = stderrors.New("invalid parameters")
)

// AIError AI 客户端错误结构
// 为了向后兼容性保留此结构，但内部使用通用错误系统
type AIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Cause   error  `json:"-"`
}

// Error 实现 error 接口
func (e *AIError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Unwrap 返回原始错误
func (e *AIError) Unwrap() error {
	return e.Cause
}

// NewAIError 创建新的 AI 错误
// 使用通用错误工厂创建错误，然后包装为 AIError 以保持向后兼容性
func NewAIError(code, message string, cause error) *AIError {
	var appErr *errors.AppError
	if cause != nil {
		appErr = errors.WrapError(code, message, cause)
	} else {
		appErr = errors.NewError(code, message)
	}

	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewAIErrorWithDetails 创建带详细信息的 AI 错误
// 使用通用错误工厂创建错误，然后包装为 AIError 以保持向后兼容性
func NewAIErrorWithDetails(code, message, details string, cause error) *AIError {
	var appErr *errors.AppError
	if cause != nil {
		appErr = errors.WrapErrorWithDetails(code, message, cause, details)
	} else {
		appErr = errors.NewErrorWithDetails(code, message, details)
	}

	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// 错误代码常量
// 使用通用错误系统中的错误代码，保持原有名称以保持向后兼容性
const (
	ErrCodeClientNotFound       = errors.ErrCodeClientNotFound
	ErrCodeInvalidConfig        = errors.ErrCodeInvalidConfig
	ErrCodeAPIKeyMissing        = errors.ErrCodeAPIKeyMissing
	ErrCodeNetworkFailed        = errors.ErrCodeNetworkFailed
	ErrCodeInvalidResponse      = errors.ErrCodeInvalidResponse
	ErrCodeToolCallFailed       = errors.ErrCodeToolCallFailed
	ErrCodeContextCanceled      = errors.ErrCodeContextCanceled
	ErrCodeTimeout              = errors.ErrCodeTimeout
	ErrCodeRateLimited          = errors.ErrCodeRateLimited
	ErrCodeModelNotSupported    = errors.ErrCodeModelNotSupported
	ErrCodeInvalidParameters    = errors.ErrCodeInvalidParameters
	ErrCodeClientCreationFailed = errors.ErrCodeClientCreationFailed
	ErrCodeForbidden            = errors.ErrCodeForbidden
	ErrCodeServiceUnavailable   = errors.ErrCodeServiceUnavailable
)

// AI特定的错误创建函数，使用通用错误工厂

// NewClientNotFoundError 创建客户端未找到错误
func NewClientNotFoundError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeClientNotFound, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewInvalidConfigError 创建无效配置错误
func NewInvalidConfigError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeInvalidConfig, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewAPIKeyMissingError 创建API密钥缺失错误
func NewAPIKeyMissingError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeAPIKeyMissing, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewNetworkFailedError 创建网络请求失败错误
func NewNetworkFailedError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeNetworkFailed, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewInvalidResponseError 创建无效响应错误
func NewInvalidResponseError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeInvalidResponse, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewToolCallFailedError 创建工具调用失败错误
func NewToolCallFailedError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeToolCallFailed, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewContextCanceledError 创建上下文取消错误
func NewContextCanceledError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeContextCanceled, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewTimeoutError 创建超时错误
func NewTimeoutError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeTimeout, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewRateLimitedError 创建速率限制错误
func NewRateLimitedError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeRateLimited, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewModelNotSupportedError 创建模型不支持错误
func NewModelNotSupportedError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeModelNotSupported, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewInvalidParametersError 创建无效参数错误
func NewInvalidParametersError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeInvalidParameters, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewClientCreationFailedError 创建客户端创建失败错误
func NewClientCreationFailedError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeClientCreationFailed, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewForbiddenError 创建禁止访问错误
func NewForbiddenError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeForbidden, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}

// NewServiceUnavailableError 创建服务不可用错误
func NewServiceUnavailableError(message string, cause error) *AIError {
	appErr := errors.WrapError(errors.ErrCodeServiceUnavailable, message, cause)
	return &AIError{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
		Cause:   appErr.Cause,
	}
}
