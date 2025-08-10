package ai

import (
	"fmt"
	"sync"
)

// AdapterRegistry 适配器注册表
type AdapterRegistry struct {
	// factories 适配器工厂函数映射
	factories map[string]AdapterFactory

	// validators 配置验证器映射
	validators map[string]ConfigValidator

	// adapters 已创建的适配器实例映射
	adapters map[string]ModelAdapter

	// adapterInfos 适配器信息映射
	adapterInfos map[string]AdapterInfo

	// mu 读写锁，保证线程安全
	mu sync.RWMutex
}

// NewAdapterRegistry 创建新的适配器注册表
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		factories:    make(map[string]AdapterFactory),
		validators:   make(map[string]ConfigValidator),
		adapters:     make(map[string]ModelAdapter),
		adapterInfos: make(map[string]AdapterInfo),
	}
}

// RegisterAdapterFactory 注册适配器工厂函数
func (r *AdapterRegistry) RegisterAdapterFactory(adapterType string, factory AdapterFactory, info AdapterInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if adapterType == "" {
		return NewAIError(ErrCodeInvalidParameters, "adapter type cannot be empty", nil)
	}

	if factory == nil {
		return NewAIError(ErrCodeInvalidParameters, "adapter factory cannot be nil", nil)
	}

	r.factories[adapterType] = factory
	r.adapterInfos[adapterType] = info

	return nil
}

// RegisterConfigValidator 注册配置验证器
func (r *AdapterRegistry) RegisterConfigValidator(adapterType string, validator ConfigValidator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if adapterType == "" {
		return NewAIError(ErrCodeInvalidParameters, "adapter type cannot be empty", nil)
	}

	if validator == nil {
		return NewAIError(ErrCodeInvalidParameters, "config validator cannot be nil", nil)
	}

	r.validators[adapterType] = validator

	return nil
}

// CreateAdapter 创建适配器实例
func (r *AdapterRegistry) CreateAdapter(name, adapterType string, config interface{}) (ModelAdapter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在同名适配器
	if _, exists := r.adapters[name]; exists {
		return nil, NewAIError(ErrCodeInvalidParameters, fmt.Sprintf("adapter with name '%s' already exists", name), nil)
	}

	// 获取适配器工厂
	factory, exists := r.factories[adapterType]
	if !exists {
		return nil, NewAIError(ErrCodeModelNotSupported, fmt.Sprintf("unsupported adapter type: %s", adapterType), nil)
	}

	// 验证配置
	if validator, exists := r.validators[adapterType]; exists {
		if err := validator(config); err != nil {
			return nil, NewAIError(ErrCodeInvalidConfig, fmt.Sprintf("invalid config for adapter type '%s'", adapterType), err)
		}
	}

	// 创建适配器实例
	adapter, err := factory(config)
	if err != nil {
		return nil, NewAIError(ErrCodeClientCreationFailed, fmt.Sprintf("failed to create adapter '%s'", name), err)
	}

	// 存储适配器实例
	r.adapters[name] = adapter

	return adapter, nil
}

// GetAdapter 获取适配器实例
func (r *AdapterRegistry) GetAdapter(name string) (ModelAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, exists := r.adapters[name]
	return adapter, exists
}

// RemoveAdapter 移除适配器实例
func (r *AdapterRegistry) RemoveAdapter(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.adapters[name]
	if !exists {
		return NewAIError(ErrCodeClientNotFound, fmt.Sprintf("adapter not found: %s", name), nil)
	}

	// 简化版本：直接移除适配器，不进行额外清理
	// 适配器的资源清理由垃圾回收器处理

	delete(r.adapters, name)
	return nil
}

// ListAdapters 列出所有已创建的适配器
func (r *AdapterRegistry) ListAdapters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name := range r.adapters {
		names = append(names, name)
	}

	return names
}

// ListSupportedTypes 列出所有支持的适配器类型
func (r *AdapterRegistry) ListSupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var types []string
	for adapterType := range r.factories {
		types = append(types, adapterType)
	}

	return types
}

// GetAdapterInfo 获取适配器类型信息
func (r *AdapterRegistry) GetAdapterInfo(adapterType string) (AdapterInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info, exists := r.adapterInfos[adapterType]
	return info, exists
}

// GetAllAdapterInfos 获取所有适配器类型信息
func (r *AdapterRegistry) GetAllAdapterInfos() map[string]AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make(map[string]AdapterInfo, len(r.adapterInfos))
	for k, v := range r.adapterInfos {
		infos[k] = v
	}

	return infos
}

// ValidateConfig 验证指定类型的配置
func (r *AdapterRegistry) ValidateConfig(adapterType string, config interface{}) error {
	r.mu.RLock()
	validator, exists := r.validators[adapterType]
	r.mu.RUnlock()

	if !exists {
		// 如果没有注册验证器，跳过验证
		return nil
	}

	return validator(config)
}

// HasAdapterType 检查是否支持指定的适配器类型
func (r *AdapterRegistry) HasAdapterType(adapterType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.factories[adapterType]
	return exists
}

// GetAdapterCount 获取已创建的适配器数量
func (r *AdapterRegistry) GetAdapterCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.adapters)
}

// GetSupportedTypesCount 获取支持的适配器类型数量
func (r *AdapterRegistry) GetSupportedTypesCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.factories)
}

// CleanupAll 清理所有适配器资源
func (r *AdapterRegistry) CleanupAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errors []error

	// 简化版本：直接清空映射，不进行额外清理
	// 适配器的资源清理由垃圾回收器处理
	_ = r.adapters // 避免未使用变量警告

	// 清空映射
	r.adapters = make(map[string]ModelAdapter)

	// 如果有错误，返回第一个错误
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// Clone 创建注册表的副本（不包含适配器实例）
func (r *AdapterRegistry) Clone() *AdapterRegistry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	newRegistry := NewAdapterRegistry()

	// 复制工厂函数
	for k, v := range r.factories {
		newRegistry.factories[k] = v
	}

	// 复制验证器
	for k, v := range r.validators {
		newRegistry.validators[k] = v
	}

	// 复制适配器信息
	for k, v := range r.adapterInfos {
		newRegistry.adapterInfos[k] = v
	}

	return newRegistry
}

// RegistryStats 注册表统计信息
type RegistryStats struct {
	// SupportedTypes 支持的适配器类型数量
	SupportedTypes int `json:"supported_types"`

	// CreatedAdapters 已创建的适配器数量
	CreatedAdapters int `json:"created_adapters"`

	// RegisteredValidators 已注册的验证器数量
	RegisteredValidators int `json:"registered_validators"`

	// AdapterNames 适配器名称列表
	AdapterNames []string `json:"adapter_names"`

	// TypeNames 适配器类型名称列表
	TypeNames []string `json:"type_names"`
}

// GetStats 获取注册表统计信息
func (r *AdapterRegistry) GetStats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var adapterNames []string
	for name := range r.adapters {
		adapterNames = append(adapterNames, name)
	}

	var typeNames []string
	for typeName := range r.factories {
		typeNames = append(typeNames, typeName)
	}

	return RegistryStats{
		SupportedTypes:       len(r.factories),
		CreatedAdapters:      len(r.adapters),
		RegisteredValidators: len(r.validators),
		AdapterNames:         adapterNames,
		TypeNames:            typeNames,
	}
}

// 全局适配器注册表实例
var defaultRegistry = NewAdapterRegistry()

// RegisterAdapterFactory 在全局注册表中注册适配器工厂函数
func RegisterAdapterFactory(adapterType string, factory AdapterFactory, info AdapterInfo) error {
	return defaultRegistry.RegisterAdapterFactory(adapterType, factory, info)
}

// RegisterConfigValidator 在全局注册表中注册配置验证器
func RegisterConfigValidator(adapterType string, validator ConfigValidator) error {
	return defaultRegistry.RegisterConfigValidator(adapterType, validator)
}

// CreateAdapter 在全局注册表中创建适配器实例
func CreateAdapter(name, adapterType string, config interface{}) (ModelAdapter, error) {
	return defaultRegistry.CreateAdapter(name, adapterType, config)
}

// GetAdapter 从全局注册表中获取适配器实例
func GetAdapter(name string) (ModelAdapter, bool) {
	return defaultRegistry.GetAdapter(name)
}

// GetDefaultRegistry 获取默认的全局注册表
func GetDefaultRegistry() *AdapterRegistry {
	return defaultRegistry
}
