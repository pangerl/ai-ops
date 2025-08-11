package errors

// DefaultErrorHandler 默认错误处理器实现
type DefaultErrorHandler struct{}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(err error) {
	// 默认实现只是打印错误
	// 在实际应用中，这里可以添加日志记录、监控告警等逻辑
	if err == nil {
		return
	}

	// 获取错误详情
	var appErr *AppError
	if ae, ok := err.(*AppError); ok {
		appErr = ae
	} else {
		// 如果不是AppError，包装一下
		appErr = WrapError(ErrCodeInternalErr, "未知错误", err)
	}

	// 简单的错误日志记录
	// 在实际应用中，这里应该使用适当的日志库
	_ = appErr // 避免未使用变量的编译错误，实际应用中应该使用日志记录
}

// GetUserFriendlyMessage 获取用户友好的错误消息
func (h *DefaultErrorHandler) GetUserFriendlyMessage(err error) string {
	if err == nil {
		return ""
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return "发生未知错误"
	}

	switch appErr.Code {
	// 系统错误
	case ErrCodeSystemError, ErrCodeInternalErr:
		return "系统错误，请联系技术支持"
	case ErrCodeInitializationFailed:
		return "应用程序初始化失败，请检查配置"
	case ErrCodeNotFound:
		return "请求的资源不存在"
	case ErrCodeInvalidParam:
		return "参数无效，请检查输入"

	// 配置错误
	case ErrCodeConfigNotFound, ErrCodeConfigInvalid, ErrCodeConfigLoadFailed, ErrCodeConfigParseFailed:
		return "配置文件错误，请检查配置文件"

	// 网络错误
	case ErrCodeNetworkFailed, ErrCodeAPIRequestFailed:
		return "网络请求失败，请检查网络连接"
	case ErrCodeTimeout:
		return "请求超时，请稍后重试"
	case ErrCodeRateLimited:
		return "请求频率过高，请稍后重试"
	case ErrCodeForbidden:
		return "访问被拒绝，请检查权限设置"
	case ErrCodeServiceUnavailable:
		return "服务暂时不可用，请稍后重试"

	// AI错误
	case ErrCodeModelNotFound, ErrCodeClientNotFound, ErrCodeModelNotSupported:
		return "AI模型配置错误，请检查配置"
	case ErrCodeAIResponseInvalid, ErrCodeInvalidResponse, ErrCodeInvalidConfig:
		return "AI服务响应错误，请稍后重试"
	case ErrCodeAPIKeyMissing:
		return "API密钥未配置，请检查配置"
	case ErrCodeToolCallFailed:
		return "AI工具调用失败，请稍后重试"
	case ErrCodeContextCanceled:
		return "请求被取消，请重试"
	case ErrCodeInvalidParameters:
		return "AI请求参数无效，请检查参数"
	case ErrCodeClientCreationFailed:
		return "AI客户端创建失败，请检查配置和网络"

	// 工具错误
	case ErrCodeToolNotFound:
		return "请求的工具不存在，请检查工具名称"
	case ErrCodeToolExecutionFailed:
		return "工具执行失败，请检查参数和输入"

	// MCP错误
	case ErrCodeMCPNotConfigured, ErrCodeMCPConnectionFailed, ErrCodeMCPNotConnected:
		return "MCP服务器连接失败，请检查配置"
	case ErrCodeMCPToolListFailed, ErrCodeMCPToolCallFailed:
		return "MCP工具调用失败，请检查工具参数和服务器状态"

	default:
		return appErr.Message
	}
}

// 默认错误处理器实例
var DefaultHandler = &DefaultErrorHandler{}

// HandleError 处理错误
func HandleError(err error) {
	DefaultHandler.HandleError(err)
}

// GetUserFriendlyMessage 获取用户友好的错误消息
func GetUserFriendlyMessage(err error) string {
	return DefaultHandler.GetUserFriendlyMessage(err)
}
