package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// 错误代码常量
const (
	// 系统级错误
	ErrCodeSystemError          = "SYSTEM_ERROR"          // 系统错误
	ErrCodeInternalErr          = "INTERNAL_ERROR"        // 内部错误
	ErrCodeInitializationFailed = "INITIALIZATION_FAILED" // 初始化失败
	ErrCodeNotFound             = "NOT_FOUND"             // 资源未找到
	ErrCodeInvalidParam         = "INVALID_PARAM"         // 无效参数

	// 配置错误
	ErrCodeConfigNotFound    = "CONFIG_NOT_FOUND"    // 配置文件未找到
	ErrCodeConfigInvalid     = "CONFIG_INVALID"      // 配置文件无效
	ErrCodeConfigLoadFailed  = "CONFIG_LOAD_FAILED"  // 配置加载失败
	ErrCodeConfigParseFailed = "CONFIG_PARSE_FAILED" // 配置解析失败

	// 网络错误
	ErrCodeNetworkFailed      = "NETWORK_FAILED"      // 网络请求失败
	ErrCodeAPIRequestFailed   = "API_REQUEST_FAILED"  // API请求失败
	ErrCodeTimeout            = "TIMEOUT"             // 请求超时
	ErrCodeRateLimited        = "RATE_LIMITED"        // 请求频率限制
	ErrCodeForbidden          = "FORBIDDEN"           // 禁止访问
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE" // 服务不可用

	// AI错误
	ErrCodeModelNotFound        = "MODEL_NOT_FOUND"        // 模型未找到
	ErrCodeAIResponseInvalid    = "AI_RESPONSE_INVALID"    // AI响应无效
	ErrCodeClientNotFound       = "CLIENT_NOT_FOUND"       // AI客户端未找到
	ErrCodeInvalidConfig        = "INVALID_CONFIG"         // 无效配置
	ErrCodeAPIKeyMissing        = "API_KEY_MISSING"        // API密钥缺失
	ErrCodeInvalidResponse      = "INVALID_RESPONSE"       // 无效响应
	ErrCodeToolCallFailed       = "TOOL_CALL_FAILED"       // 工具调用失败
	ErrCodeContextCanceled      = "CONTEXT_CANCELED"       // 上下文取消
	ErrCodeModelNotSupported    = "MODEL_NOT_SUPPORTED"    // 模型不支持
	ErrCodeInvalidParameters    = "INVALID_PARAMETERS"     // 无效参数
	ErrCodeClientCreationFailed = "CLIENT_CREATION_FAILED" // 客户端创建失败

	// 工具错误
	ErrCodeToolNotFound        = "TOOL_NOT_FOUND"        // 工具未找到
	ErrCodeToolExecutionFailed = "TOOL_EXECUTION_FAILED" // 工具执行失败

	// MCP错误
	ErrCodeMCPNotConfigured    = "MCP_NOT_CONFIGURED"    // MCP未配置
	ErrCodeMCPConnectionFailed = "MCP_CONNECTION_FAILED" // MCP连接失败
	ErrCodeMCPNotConnected     = "MCP_NOT_CONNECTED"     // MCP未连接
	ErrCodeMCPToolListFailed   = "MCP_TOOL_LIST_FAILED"  // MCP工具列表获取失败
	ErrCodeMCPToolCallFailed   = "MCP_TOOL_CALL_FAILED"  // MCP工具调用失败
)

// AppError 应用错误结构
type AppError struct {
	Code    string `json:"code"`              // 错误代码
	Message string `json:"message"`           // 错误消息
	Details string `json:"details,omitempty"` // 错误详情
	Cause   error  `json:"-"`                 // 原始错误
	Stack   string `json:"stack,omitempty"`   // 错误堆栈
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is 实现错误比较接口
func (e *AppError) Is(target error) bool {
	if other, ok := target.(*AppError); ok {
		return e.Code == other.Code
	}
	return false
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithStack 添加堆栈信息
func (e *AppError) WithStack() *AppError {
	e.Stack = getStackTrace(3) // 跳过3层调用栈
	return e
}

// getStackTrace 获取调用堆栈
func getStackTrace(skip int) string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var stack strings.Builder
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		// 跳过runtime相关的调用栈
		if strings.Contains(frame.File, "runtime/") {
			continue
		}
		stack.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))
	}
	return stack.String()
}

// IsErrorCode 检查错误是否为指定类型
func IsErrorCode(err error, code string) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// GetErrorCode 获取错误代码
func GetErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternalErr
}

// GetErrorDetails 获取错误详情
func GetErrorDetails(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Details
	}
	return ""
}
