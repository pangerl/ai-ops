package llm

import (
	"ai-ops/internal/util/errors"
	"strings"
)

// ErrorMapper 错误映射器接口
type ErrorMapper interface {
	// MapError 将原始错误映射为标准化的AI错误
	MapError(originalError error) error
}

// ErrorMappingRule 错误映射规则
type ErrorMappingRule struct {
	// Pattern 匹配模式（用于字符串包含匹配）
	Pattern string `json:"pattern"`

	// ErrorCode 目标错误代码
	ErrorCode string `json:"error_code"`

	// ErrorMessage 目标错误消息
	ErrorMessage string `json:"error_message"`
}

// DefaultErrorMapper 默认错误映射器实现
type DefaultErrorMapper struct {
	rules []ErrorMappingRule
}

// NewDefaultErrorMapper 创建默认错误映射器
func NewDefaultErrorMapper() *DefaultErrorMapper {
	mapper := &DefaultErrorMapper{
		rules: make([]ErrorMappingRule, 0),
	}
	mapper.addPredefinedRules()
	return mapper
}

// MapError 将原始错误映射为标准化的AI错误
func (m *DefaultErrorMapper) MapError(originalError error) error {
	if originalError == nil {
		return nil
	}

	if appErr, ok := originalError.(*errors.AppError); ok {
		return appErr
	}

	errorMessage := strings.ToLower(originalError.Error())

	for _, rule := range m.rules {
		if strings.Contains(errorMessage, strings.ToLower(rule.Pattern)) {
			return errors.WrapErrorWithDetails(rule.ErrorCode, rule.ErrorMessage, originalError, originalError.Error())
		}
	}

	return errors.WrapError(errors.ErrCodeInvalidResponse, "unknown error", originalError)
}

// addPredefinedRules 添加预定义的通用错误映射规则
func (m *DefaultErrorMapper) addPredefinedRules() {
	predefinedRules := []ErrorMappingRule{
		{Pattern: "timeout", ErrorCode: errors.ErrCodeTimeout, ErrorMessage: "Request timeout"},
		{Pattern: "context deadline exceeded", ErrorCode: errors.ErrCodeTimeout, ErrorMessage: "Request timeout"},
		{Pattern: "network", ErrorCode: errors.ErrCodeNetworkFailed, ErrorMessage: "Network request failed"},
		{Pattern: "connection", ErrorCode: errors.ErrCodeNetworkFailed, ErrorMessage: "Connection failed"},
		{Pattern: "rate limit", ErrorCode: errors.ErrCodeRateLimited, ErrorMessage: "Rate limit exceeded"},
		{Pattern: "too many requests", ErrorCode: errors.ErrCodeRateLimited, ErrorMessage: "Rate limit exceeded"},
		{Pattern: "unauthorized", ErrorCode: errors.ErrCodeAPIKeyMissing, ErrorMessage: "Authentication failed"},
		{Pattern: "forbidden", ErrorCode: errors.ErrCodeForbidden, ErrorMessage: "Access forbidden"},
		{Pattern: "invalid api key", ErrorCode: errors.ErrCodeAPIKeyMissing, ErrorMessage: "Invalid API key"},
		{Pattern: "bad request", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "Invalid request parameters"},
		{Pattern: "invalid parameter", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "Invalid parameters"},
		{Pattern: "model not found", ErrorCode: errors.ErrCodeModelNotSupported, ErrorMessage: "Model not found or not supported"},
		{Pattern: "server error", ErrorCode: errors.ErrCodeNetworkFailed, ErrorMessage: "Server error"},
		{Pattern: "internal error", ErrorCode: errors.ErrCodeNetworkFailed, ErrorMessage: "Internal server error"},
	}
	m.rules = append(m.rules, predefinedRules...)
}

// ProviderSpecificErrorMapper 提供商特定的错误映射器
type ProviderSpecificErrorMapper struct {
	*DefaultErrorMapper
}

// NewProviderSpecificErrorMapper 创建提供商特定的错误映射器
func NewProviderSpecificErrorMapper(provider string) *ProviderSpecificErrorMapper {
	mapper := &ProviderSpecificErrorMapper{
		DefaultErrorMapper: NewDefaultErrorMapper(),
	}
	mapper.addProviderSpecificRules(provider)
	return mapper
}

// addProviderSpecificRules 添加提供商特定的错误映射规则
func (m *ProviderSpecificErrorMapper) addProviderSpecificRules(provider string) {
	var specificRules []ErrorMappingRule

	switch strings.ToLower(provider) {
	case "openai":
		specificRules = []ErrorMappingRule{
			{Pattern: "insufficient_quota", ErrorCode: errors.ErrCodeRateLimited, ErrorMessage: "OpenAI quota exceeded"},
			{Pattern: "model_not_found", ErrorCode: errors.ErrCodeModelNotSupported, ErrorMessage: "OpenAI model not found"},
			{Pattern: "invalid_request_error", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "OpenAI invalid request"},
		}
	case "gemini":
		specificRules = []ErrorMappingRule{
			{Pattern: "SAFETY", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "Gemini safety violation"},
			{Pattern: "RECITATION", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "Gemini recitation detected"},
			{Pattern: "blocked", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "Gemini request blocked"},
		}
	case "claude":
		specificRules = []ErrorMappingRule{
			{Pattern: "overloaded_error", ErrorCode: errors.ErrCodeRateLimited, ErrorMessage: "Claude service overloaded"},
			{Pattern: "invalid_request_error", ErrorCode: errors.ErrCodeInvalidParameters, ErrorMessage: "Claude invalid request"},
		}
	}

	// 将特定规则添加到规则列表的开头，以便优先匹配
	m.rules = append(specificRules, m.rules...)
}

// CreateErrorMapperForProvider 为指定提供商创建错误映射器
func CreateErrorMapperForProvider(provider string) ErrorMapper {
	if provider == "" {
		return NewDefaultErrorMapper()
	}
	return NewProviderSpecificErrorMapper(provider)
}
