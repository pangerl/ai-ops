package errors

// NewError 创建新的错误
func NewError(code, message string) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
	}
	return err.WithStack()
}

// NewErrorWithDetails 创建带详情的错误
func NewErrorWithDetails(code, message, details string) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
	return err.WithStack()
}

// WrapError 包装现有错误
func WrapError(code, message string, cause error) *AppError {
	details := ""
	if cause != nil {
		details = cause.Error()
	}

	err := &AppError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
	return err.WithStack()
}

// WrapErrorWithDetails 包装现有错误并添加详情
func WrapErrorWithDetails(code, message string, cause error, details string) *AppError {
	err := &AppError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
	return err.WithStack()
}

// 预定义错误创建函数

// 系统错误
func NewSystemError(message string) *AppError {
	return NewError(ErrCodeSystemError, message)
}

func NewSystemErrorWithDetails(message, details string) *AppError {
	return NewErrorWithDetails(ErrCodeSystemError, message, details)
}

func WrapSystemError(message string, cause error) *AppError {
	return WrapError(ErrCodeSystemError, message, cause)
}

// 配置错误
func NewConfigError(message string) *AppError {
	return NewError(ErrCodeConfigInvalid, message)
}

func NewConfigErrorWithDetails(message, details string) *AppError {
	return NewErrorWithDetails(ErrCodeConfigInvalid, message, details)
}

func WrapConfigError(message string, cause error) *AppError {
	return WrapError(ErrCodeConfigInvalid, message, cause)
}

// 网络错误
func NewNetworkError(message string) *AppError {
	return NewError(ErrCodeNetworkFailed, message)
}

func NewNetworkErrorWithDetails(message, details string) *AppError {
	return NewErrorWithDetails(ErrCodeNetworkFailed, message, details)
}

func WrapNetworkError(message string, cause error) *AppError {
	return WrapError(ErrCodeNetworkFailed, message, cause)
}

// AI错误
func NewAIError(message string) *AppError {
	return NewError(ErrCodeAIResponseInvalid, message)
}

func NewAIErrorWithDetails(message, details string) *AppError {
	return NewErrorWithDetails(ErrCodeAIResponseInvalid, message, details)
}

func WrapAIError(message string, cause error) *AppError {
	return WrapError(ErrCodeAIResponseInvalid, message, cause)
}

// 工具错误
func NewToolError(message string) *AppError {
	return NewError(ErrCodeToolExecutionFailed, message)
}

func NewToolErrorWithDetails(message, details string) *AppError {
	return NewErrorWithDetails(ErrCodeToolExecutionFailed, message, details)
}

func WrapToolError(message string, cause error) *AppError {
	return WrapError(ErrCodeToolExecutionFailed, message, cause)
}

// MCP错误
func NewMCPError(message string) *AppError {
	return NewError(ErrCodeMCPConnectionFailed, message)
}

func NewMCPErrorWithDetails(message, details string) *AppError {
	return NewErrorWithDetails(ErrCodeMCPConnectionFailed, message, details)
}

func WrapMCPError(message string, cause error) *AppError {
	return WrapError(ErrCodeMCPConnectionFailed, message, cause)
}
