package ai

import (
	"ai-ops/internal/common/errors"
	"regexp"
	"strings"
)

// ErrorMapper 错误映射器接口
type ErrorMapper interface {
	// MapError 将原始错误映射为标准化的AI错误
	MapError(originalError error) error

	// AddMapping 添加错误映射规则
	AddMapping(rule ErrorMappingRule)

	// RemoveMapping 移除错误映射规则
	RemoveMapping(pattern string)

	// GetMappings 获取所有映射规则
	GetMappings() []ErrorMappingRule
}

// ErrorMappingRule 错误映射规则
type ErrorMappingRule struct {
	// Pattern 匹配模式（正则表达式）
	Pattern string `json:"pattern"`

	// ErrorCode 目标错误代码
	ErrorCode string `json:"error_code"`

	// ErrorMessage 目标错误消息
	ErrorMessage string `json:"error_message"`

	// Priority 优先级（数字越大优先级越高）
	Priority int `json:"priority"`

	// CaseSensitive 是否大小写敏感
	CaseSensitive bool `json:"case_sensitive"`

	// MatchType 匹配类型：contains, regex, exact
	MatchType string `json:"match_type"`
}

// DefaultErrorMapper 默认错误映射器实现
type DefaultErrorMapper struct {
	rules       []ErrorMappingRule
	compiledMap map[string]*regexp.Regexp
}

// NewDefaultErrorMapper 创建默认错误映射器
func NewDefaultErrorMapper() *DefaultErrorMapper {
	mapper := &DefaultErrorMapper{
		rules:       make([]ErrorMappingRule, 0),
		compiledMap: make(map[string]*regexp.Regexp),
	}

	// 添加预定义的通用错误映射规则
	mapper.addPredefinedRules()

	return mapper
}

// MapError 将原始错误映射为标准化的AI错误
func (m *DefaultErrorMapper) MapError(originalError error) error {
	if originalError == nil {
		return nil
	}

	// 如果已经是AppError，检查是否需要重新映射
	if appErr, ok := originalError.(*errors.AppError); ok {
		// 可以根据需要对已有的AppError进行进一步映射
		return appErr
	}

	// 如果是AIError，转换为AppError
	if aiErr, ok := originalError.(*AIError); ok {
		// 将AIError转换为AppError
		if aiErr.Cause != nil {
			return errors.WrapErrorWithDetails(aiErr.Code, aiErr.Message, aiErr.Cause, aiErr.Details)
		}
		return errors.NewErrorWithDetails(aiErr.Code, aiErr.Message, aiErr.Details)
	}

	errorMessage := originalError.Error()

	// 按优先级排序规则，优先级高的优先匹配
	for _, rule := range m.getSortedRules() {
		if m.matchRule(errorMessage, rule) {
			return errors.WrapErrorWithDetails(rule.ErrorCode, rule.ErrorMessage, originalError, errorMessage)
		}
	}

	// 如果没有匹配的规则，返回通用错误
	return errors.WrapError(errors.ErrCodeInvalidResponse, "unknown error", originalError)
}

// AddMapping 添加错误映射规则
func (m *DefaultErrorMapper) AddMapping(rule ErrorMappingRule) {
	m.rules = append(m.rules, rule)

	// 如果是正则表达式，预编译
	if rule.MatchType == "regex" {
		pattern := rule.Pattern
		if !rule.CaseSensitive {
			pattern = "(?i)" + pattern
		}

		if compiled, err := regexp.Compile(pattern); err == nil {
			m.compiledMap[rule.Pattern] = compiled
		}
	}
}

// RemoveMapping 移除错误映射规则
func (m *DefaultErrorMapper) RemoveMapping(pattern string) {
	for i, rule := range m.rules {
		if rule.Pattern == pattern {
			// 移除规则
			m.rules = append(m.rules[:i], m.rules[i+1:]...)

			// 移除编译的正则表达式
			delete(m.compiledMap, pattern)
			break
		}
	}
}

// GetMappings 获取所有映射规则
func (m *DefaultErrorMapper) GetMappings() []ErrorMappingRule {
	// 返回规则的副本
	rules := make([]ErrorMappingRule, len(m.rules))
	copy(rules, m.rules)
	return rules
}

// matchRule 检查错误消息是否匹配规则
func (m *DefaultErrorMapper) matchRule(errorMessage string, rule ErrorMappingRule) bool {
	message := errorMessage
	pattern := rule.Pattern

	// 处理大小写敏感性
	if !rule.CaseSensitive {
		message = strings.ToLower(message)
		pattern = strings.ToLower(pattern)
	}

	switch rule.MatchType {
	case "exact":
		return message == pattern
	case "contains":
		return strings.Contains(message, pattern)
	case "regex":
		if compiled, exists := m.compiledMap[rule.Pattern]; exists {
			return compiled.MatchString(message)
		}
		// 如果编译失败，降级为包含匹配
		return strings.Contains(message, pattern)
	default:
		// 默认使用包含匹配
		return strings.Contains(message, pattern)
	}
}

// getSortedRules 获取按优先级排序的规则列表
func (m *DefaultErrorMapper) getSortedRules() []ErrorMappingRule {
	rules := make([]ErrorMappingRule, len(m.rules))
	copy(rules, m.rules)

	// 按优先级降序排序
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	return rules
}

// addPredefinedRules 添加预定义的通用错误映射规则
func (m *DefaultErrorMapper) addPredefinedRules() {
	predefinedRules := []ErrorMappingRule{
		{
			Pattern:       "timeout",
			ErrorCode:     errors.ErrCodeTimeout,
			ErrorMessage:  "Request timeout",
			Priority:      100,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "context deadline exceeded",
			ErrorCode:     errors.ErrCodeTimeout,
			ErrorMessage:  "Request timeout",
			Priority:      95,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "network",
			ErrorCode:     errors.ErrCodeNetworkFailed,
			ErrorMessage:  "Network request failed",
			Priority:      90,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "connection",
			ErrorCode:     errors.ErrCodeNetworkFailed,
			ErrorMessage:  "Connection failed",
			Priority:      85,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "rate limit",
			ErrorCode:     errors.ErrCodeRateLimited,
			ErrorMessage:  "Rate limit exceeded",
			Priority:      80,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "too many requests",
			ErrorCode:     errors.ErrCodeRateLimited,
			ErrorMessage:  "Rate limit exceeded",
			Priority:      75,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "unauthorized",
			ErrorCode:     errors.ErrCodeAPIKeyMissing,
			ErrorMessage:  "Authentication failed",
			Priority:      70,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "forbidden",
			ErrorCode:     errors.ErrCodeForbidden,
			ErrorMessage:  "Access forbidden",
			Priority:      65,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "invalid.*api.*key",
			ErrorCode:     errors.ErrCodeAPIKeyMissing,
			ErrorMessage:  "Invalid API key",
			Priority:      60,
			CaseSensitive: false,
			MatchType:     "regex",
		},
		{
			Pattern:       "bad request",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "Invalid request parameters",
			Priority:      55,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "invalid.*parameter",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "Invalid parameters",
			Priority:      50,
			CaseSensitive: false,
			MatchType:     "regex",
		},
		{
			Pattern:       "model.*not.*found",
			ErrorCode:     errors.ErrCodeModelNotSupported,
			ErrorMessage:  "Model not found or not supported",
			Priority:      45,
			CaseSensitive: false,
			MatchType:     "regex",
		},
		{
			Pattern:       "server error",
			ErrorCode:     errors.ErrCodeNetworkFailed,
			ErrorMessage:  "Server error",
			Priority:      40,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "internal.*error",
			ErrorCode:     errors.ErrCodeNetworkFailed,
			ErrorMessage:  "Internal server error",
			Priority:      35,
			CaseSensitive: false,
			MatchType:     "regex",
		},
	}

	for _, rule := range predefinedRules {
		m.AddMapping(rule)
	}
}

// ProviderSpecificErrorMapper 提供商特定的错误映射器
type ProviderSpecificErrorMapper struct {
	*DefaultErrorMapper
	provider string
}

// NewProviderSpecificErrorMapper 创建提供商特定的错误映射器
func NewProviderSpecificErrorMapper(provider string) *ProviderSpecificErrorMapper {
	mapper := &ProviderSpecificErrorMapper{
		DefaultErrorMapper: NewDefaultErrorMapper(),
		provider:           provider,
	}

	// 添加提供商特定的规则
	mapper.addProviderSpecificRules()

	return mapper
}

// addProviderSpecificRules 添加提供商特定的错误映射规则
func (m *ProviderSpecificErrorMapper) addProviderSpecificRules() {
	switch strings.ToLower(m.provider) {
	case "openai":
		m.addOpenAIRules()
	case "gemini":
		m.addGeminiRules()
	case "claude":
		m.addClaudeRules()
	}
}

// addOpenAIRules 添加OpenAI特定的错误映射规则
func (m *ProviderSpecificErrorMapper) addOpenAIRules() {
	rules := []ErrorMappingRule{
		{
			Pattern:       "insufficient_quota",
			ErrorCode:     errors.ErrCodeRateLimited,
			ErrorMessage:  "OpenAI quota exceeded",
			Priority:      200,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "model_not_found",
			ErrorCode:     errors.ErrCodeModelNotSupported,
			ErrorMessage:  "OpenAI model not found",
			Priority:      195,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "invalid_request_error",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "OpenAI invalid request",
			Priority:      190,
			CaseSensitive: false,
			MatchType:     "contains",
		},
	}

	for _, rule := range rules {
		m.AddMapping(rule)
	}
}

// addGeminiRules 添加Gemini特定的错误映射规则
func (m *ProviderSpecificErrorMapper) addGeminiRules() {
	rules := []ErrorMappingRule{
		{
			Pattern:       "SAFETY",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "Gemini safety violation",
			Priority:      200,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "RECITATION",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "Gemini recitation detected",
			Priority:      195,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "blocked",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "Gemini request blocked",
			Priority:      190,
			CaseSensitive: false,
			MatchType:     "contains",
		},
	}

	for _, rule := range rules {
		m.AddMapping(rule)
	}
}

// addClaudeRules 添加Claude特定的错误映射规则
func (m *ProviderSpecificErrorMapper) addClaudeRules() {
	rules := []ErrorMappingRule{
		{
			Pattern:       "overloaded_error",
			ErrorCode:     errors.ErrCodeRateLimited,
			ErrorMessage:  "Claude service overloaded",
			Priority:      200,
			CaseSensitive: false,
			MatchType:     "contains",
		},
		{
			Pattern:       "invalid_request_error",
			ErrorCode:     errors.ErrCodeInvalidParameters,
			ErrorMessage:  "Claude invalid request",
			Priority:      195,
			CaseSensitive: false,
			MatchType:     "contains",
		},
	}

	for _, rule := range rules {
		m.AddMapping(rule)
	}
}

// ChainErrorMapper 错误映射器链
type ChainErrorMapper struct {
	mappers []ErrorMapper
}

// NewChainErrorMapper 创建错误映射器链
func NewChainErrorMapper(mappers ...ErrorMapper) *ChainErrorMapper {
	return &ChainErrorMapper{
		mappers: mappers,
	}
}

// MapError 使用链中的映射器依次处理错误
func (c *ChainErrorMapper) MapError(originalError error) error {
	currentError := originalError

	for _, mapper := range c.mappers {
		currentError = mapper.MapError(currentError)
	}

	return currentError
}

// AddMapping 向链中的所有映射器添加规则
func (c *ChainErrorMapper) AddMapping(rule ErrorMappingRule) {
	for _, mapper := range c.mappers {
		mapper.AddMapping(rule)
	}
}

// RemoveMapping 从链中的所有映射器移除规则
func (c *ChainErrorMapper) RemoveMapping(pattern string) {
	for _, mapper := range c.mappers {
		mapper.RemoveMapping(pattern)
	}
}

// GetMappings 获取链中第一个映射器的规则
func (c *ChainErrorMapper) GetMappings() []ErrorMappingRule {
	if len(c.mappers) > 0 {
		return c.mappers[0].GetMappings()
	}
	return []ErrorMappingRule{}
}

// ErrorMappingStats 错误映射统计
type ErrorMappingStats struct {
	// TotalMappings 总映射规则数
	TotalMappings int `json:"total_mappings"`

	// MappingsByProvider 按提供商分组的映射数
	MappingsByProvider map[string]int `json:"mappings_by_provider"`

	// MappingsByErrorCode 按错误代码分组的映射数
	MappingsByErrorCode map[string]int `json:"mappings_by_error_code"`

	// MostFrequentPatterns 最常见的错误模式
	MostFrequentPatterns []string `json:"most_frequent_patterns"`
}

// GetErrorMappingStats 获取错误映射统计信息
func GetErrorMappingStats(mapper ErrorMapper) ErrorMappingStats {
	mappings := mapper.GetMappings()

	stats := ErrorMappingStats{
		TotalMappings:        len(mappings),
		MappingsByProvider:   make(map[string]int),
		MappingsByErrorCode:  make(map[string]int),
		MostFrequentPatterns: make([]string, 0),
	}

	// 统计按错误代码分组
	for _, mapping := range mappings {
		stats.MappingsByErrorCode[mapping.ErrorCode]++
	}

	return stats
}

// CreateErrorMapperForProvider 为指定提供商创建错误映射器
func CreateErrorMapperForProvider(provider string) ErrorMapper {
	if provider == "" {
		return NewDefaultErrorMapper()
	}

	return NewProviderSpecificErrorMapper(provider)
}
