package util

import (
	"fmt"
)

// 错误代码常量
const (
	ErrCodeConfigNotFound       = "CONFIG_NOT_FOUND"      // 配置文件未找到
	ErrCodeConfigInvalid        = "CONFIG_INVALID"        // 配置文件无效
	ErrCodeConfigLoadFailed     = "CONFIG_LOAD_FAILED"    // 配置加载失败
	ErrCodeConfigParseFailed    = "CONFIG_PARSE_FAILED"   // 配置解析失败
	ErrCodeModelNotFound        = "MODEL_NOT_FOUND"       // 模型未找到
	ErrCodeAPIKeyMissing        = "API_KEY_MISSING"       // API密钥缺失
	ErrCodeNetworkFailed        = "NETWORK_FAILED"        // 网络请求失败
	ErrCodeAPIRequestFailed     = "API_REQUEST_FAILED"    // API请求失败
	ErrCodeToolNotFound         = "TOOL_NOT_FOUND"        // 工具未找到
	ErrCodeToolExecutionFailed  = "TOOL_EXECUTION_FAILED" // 工具执行失败
	ErrCodeInvalidParam         = "INVALID_PARAM"         // 无效参数
	ErrCodeInternalErr          = "INTERNAL_ERROR"        // 内部错误
	ErrCodeAIResponseInvalid    = "AI_RESPONSE_INVALID"   // AI响应无效
	ErrCodeNotFound             = "NOT_FOUND"             // 资源未找到
	ErrCodeInitializationFailed = "INITIALIZATION_FAILED" // 初始化失败

	// MCP相关错误代码
	ErrCodeMCPNotConfigured    = "MCP_NOT_CONFIGURED"    // MCP未配置
	ErrCodeMCPConnectionFailed = "MCP_CONNECTION_FAILED" // MCP连接失败
	ErrCodeMCPNotConnected     = "MCP_NOT_CONNECTED"     // MCP未连接
	ErrCodeMCPToolListFailed   = "MCP_TOOL_LIST_FAILED"  // MCP工具列表获取失败
	ErrCodeMCPToolCallFailed   = "MCP_TOOL_CALL_FAILED"  // MCP工具调用失败
)

// 应用错误结构
type AppError struct {
	Code    string `json:"code"`              // 错误代码
	Message string `json:"message"`           // 错误消息
	Details string `json:"details,omitempty"` // 错误详情
	Cause   error  `json:"-"`                 // 原始错误
}

// 实现error接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 获取原始错误
func (e *AppError) Unwrap() error {
	return e.Cause
}

// 创建新的应用错误
func NewError(code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// 创建带详情的应用错误
func NewErrorWithDetail(code, message, details string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// 包装现有错误
func WrapError(code, message string, cause error) *AppError {
	details := ""
	if cause != nil {
		details = cause.Error()
	}

	return &AppError{
		Code:    code,
		Message: message,
		Details: details,
		Cause:   cause,
	}
}

// 检查错误是否为指定类型
func IsErrorCode(err error, code string) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code == code
	}
	return false
}

// 获取错误代码
func GetErrorCode(err error) string {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return ErrCodeInternalErr
}

// 获取用户友好的错误消息
func GetUserFriendlyMessage(err error) string {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Code {
		case ErrCodeConfigNotFound:
			return "配置文件未找到，请检查配置文件路径"
		case ErrCodeConfigInvalid:
			return "配置文件格式错误，请检查TOML语法"
		case ErrCodeAPIKeyMissing:
			return "API密钥未配置，请在配置文件或环境变量中设置"
		case ErrCodeNetworkFailed:
			return "网络连接失败，请检查网络连接和API地址"
		case ErrCodeModelNotFound:
			return "指定的AI模型未配置，请检查配置文件"
		case ErrCodeToolNotFound:
			return "请求的工具不存在，请检查工具名称"
		case ErrCodeInvalidParam:
			return "参数无效，请检查输入参数"
		case ErrCodeMCPNotConfigured:
			return "MCP未配置，请检查mcp_settings.json文件"
		case ErrCodeMCPConnectionFailed:
			return "MCP服务器连接失败，请检查服务器配置和状态"
		case ErrCodeMCPNotConnected:
			return "MCP服务器未连接，请检查连接状态"
		case ErrCodeMCPToolCallFailed:
			return "MCP工具调用失败，请检查工具参数和服务器状态"
		default:
			return appErr.Message
		}
	}
	return "发生未知错误"
}

// 错误恢复建议
func GetRecoverySuggestion(err error) string {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Code {
		case ErrCodeConfigNotFound:
			return "运行程序会自动创建默认配置文件，或手动创建config.toml文件"
		case ErrCodeConfigInvalid:
			return "使用TOML验证工具检查配置文件语法，或重新生成默认配置"
		case ErrCodeAPIKeyMissing:
			return "在配置文件中设置api_key，或设置对应的环境变量"
		case ErrCodeNetworkFailed:
			return "检查网络连接，确认API服务地址正确，或稍后重试"
		case ErrCodeModelNotFound:
			return "在配置文件的[ai.models]部分添加模型配置"
		case ErrCodeToolNotFound:
			return "检查工具名称拼写，或查看可用工具列表"
		case ErrCodeMCPNotConfigured:
			return "创建mcp_settings.json配置文件，配置MCP服务器信息"
		case ErrCodeMCPConnectionFailed:
			return "检查MCP服务器命令和参数，确保服务器程序可执行"
		case ErrCodeMCPNotConnected:
			return "重新连接MCP服务器，或检查服务器进程状态"
		case ErrCodeMCPToolCallFailed:
			return "检查工具参数格式，或查看MCP服务器日志"
		default:
			return "查看详细错误信息或联系技术支持"
		}
	}
	return "查看错误日志获取更多信息"
}
