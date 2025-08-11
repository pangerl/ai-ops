package errors

import (
	"net/http"
)

// HTTPMiddleware HTTP错误处理中间件
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// 处理panic
				var appErr *AppError
				switch e := err.(type) {
				case *AppError:
					appErr = e
				case error:
					appErr = WrapError(ErrCodeSystemError, "系统发生panic", e)
				default:
					appErr = NewErrorWithDetails(ErrCodeSystemError, "系统发生panic", "未知错误")
				}

				// 处理错误
				DefaultHandler.HandleError(appErr)

				// 返回错误响应
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(getHTTPStatus(appErr.Code))

				// 在实际应用中，这里应该返回JSON格式的错误响应
				// 例如：json.NewEncoder(w).Encode(ErrorResponse{Code: appErr.Code, Message: appErr.Message})
				_, _ = w.Write([]byte(DefaultHandler.GetUserFriendlyMessage(appErr)))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// getHTTPStatus 根据错误代码获取HTTP状态码
func getHTTPStatus(code string) int {
	switch code {
	// 客户端错误
	case ErrCodeInvalidParam, ErrCodeInvalidParameters:
		return http.StatusBadRequest
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeAPIKeyMissing, ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests

	// 服务端错误
	case ErrCodeSystemError, ErrCodeInternalErr, ErrCodeInitializationFailed:
		return http.StatusInternalServerError
	case ErrCodeConfigNotFound, ErrCodeConfigInvalid, ErrCodeConfigLoadFailed, ErrCodeConfigParseFailed:
		return http.StatusInternalServerError
	case ErrCodeNetworkFailed, ErrCodeAPIRequestFailed:
		return http.StatusBadGateway
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeServiceUnavailable:
		return http.StatusServiceUnavailable

	// AI错误
	case ErrCodeModelNotFound, ErrCodeClientNotFound, ErrCodeModelNotSupported:
		return http.StatusBadRequest
	case ErrCodeAIResponseInvalid, ErrCodeInvalidResponse, ErrCodeInvalidConfig:
		return http.StatusInternalServerError
	case ErrCodeToolCallFailed:
		return http.StatusInternalServerError

	// 工具错误
	case ErrCodeToolNotFound:
		return http.StatusNotFound
	case ErrCodeToolExecutionFailed:
		return http.StatusInternalServerError

	// MCP错误
	case ErrCodeMCPNotConfigured, ErrCodeMCPConnectionFailed, ErrCodeMCPNotConnected:
		return http.StatusInternalServerError
	case ErrCodeMCPToolListFailed, ErrCodeMCPToolCallFailed:
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}
