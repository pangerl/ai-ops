package ai

import "errors"

// AI 客户端相关错误定义
var (
	ErrClientNotFound    = errors.New("AI client not found")
	ErrInvalidConfig     = errors.New("invalid AI client configuration")
	ErrAPIKeyMissing     = errors.New("API key is missing")
	ErrNetworkFailed     = errors.New("network request failed")
	ErrInvalidResponse   = errors.New("invalid response from AI service")
	ErrToolCallFailed    = errors.New("tool call execution failed")
	ErrContextCanceled   = errors.New("request context was canceled")
	ErrTimeout           = errors.New("request timeout")
	ErrRateLimited       = errors.New("rate limit exceeded")
	ErrModelNotSupported = errors.New("model not supported")
	ErrInvalidParameters = errors.New("invalid parameters")
)

// AIError AI 客户端错误结构
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
func NewAIError(code, message string, cause error) *AIError {
	return &AIError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewAIErrorWithDetails 创建带详细信息的 AI 错误
func NewAIErrorWithDetails(code, message, details string, cause error) *AIError {
	return &AIError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
}

// 错误代码常量
const (
	ErrCodeClientNotFound       = "CLIENT_NOT_FOUND"
	ErrCodeInvalidConfig        = "INVALID_CONFIG"
	ErrCodeAPIKeyMissing        = "API_KEY_MISSING"
	ErrCodeNetworkFailed        = "NETWORK_FAILED"
	ErrCodeInvalidResponse      = "INVALID_RESPONSE"
	ErrCodeToolCallFailed       = "TOOL_CALL_FAILED"
	ErrCodeContextCanceled      = "CONTEXT_CANCELED"
	ErrCodeTimeout              = "TIMEOUT"
	ErrCodeRateLimited          = "RATE_LIMITED"
	ErrCodeModelNotSupported    = "MODEL_NOT_SUPPORTED"
	ErrCodeInvalidParameters    = "INVALID_PARAMETERS"
	ErrCodeClientCreationFailed = "CLIENT_CREATION_FAILED"
)
