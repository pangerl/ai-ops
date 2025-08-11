package ai

import (
	"ai-ops/internal/common/errors"
	"ai-ops/internal/common/registry"
	"fmt"
	"sync"
	"time"
)

// AdapterItem 适配器注册表项，实现 RegistryItem 接口
type AdapterItem struct {
	id          string
	name        string
	adapterType string
	adapter     ModelAdapter
	info        AdapterInfo
	factory     AdapterFactory
	validator   ConfigValidator
	createdAt   time.Time
	updatedAt   time.Time
}

// ID 返回适配器项的唯一标识符
func (a *AdapterItem) ID() string {
	return a.id
}

// Name 返回适配器项的名称
func (a *AdapterItem) Name() string {
	return a.name
}

// Type 返回适配器项的类型
func (a *AdapterItem) Type() string {
	return a.adapterType
}

// AdapterRegistry 适配器注册表，基于简化的通用注册表实现
type AdapterRegistry struct {
	// baseRegistry 基础注册表实现
	baseRegistry *registry.BaseRegistry[*AdapterItem]

	// factories 适配器工厂函数映射（保留用于向后兼容）
	factories map[string]AdapterFactory

	// validators 配置验证器映射（保留用于向后兼容）
	validators map[string]ConfigValidator

	// mu 读写锁，保证线程安全
	mu sync.RWMutex
}

// NewAdapterRegistry 创建新的适配器注册表
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		baseRegistry: registry.NewBaseRegistry[*AdapterItem](),
		factories:    make(map[string]AdapterFactory),
		validators:   make(map[string]ConfigValidator),
	}
}

// RegisterAdapterFactory 注册适配器工厂函数
func (r *AdapterRegistry) RegisterAdapterFactory(adapterType string, factory AdapterFactory, info AdapterInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if adapterType == "" {
		return NewInvalidParametersError("adapter type cannot be empty", nil)
	}

	if factory == nil {
		return NewInvalidParametersError("adapter factory cannot be nil", nil)
	}

	// 保存到旧映射中（向后兼容）
	r.factories[adapterType] = factory

	// 创建适配器类型项并注册到基础注册表
	item := &AdapterItem{
		id:          "factory:" + adapterType,
		name:        info.Name,
		adapterType: adapterType,
		factory:     factory,
		info:        info,
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
	}

	if err := r.baseRegistry.Register(item); err != nil {
		return errors.WrapError(errors.ErrCodeInternalErr, "failed to register adapter factory", err)
	}

	return nil
}

// RegisterConfigValidator 注册配置验证器
func (r *AdapterRegistry) RegisterConfigValidator(adapterType string, validator ConfigValidator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if adapterType == "" {
		return NewInvalidParametersError("adapter type cannot be empty", nil)
	}

	if validator == nil {
		return NewInvalidParametersError("config validator cannot be nil", nil)
	}

	// 保存到旧映射中（向后兼容）
	r.validators[adapterType] = validator

	// 检查是否已存在对应的工厂项
	itemID := "factory:" + adapterType
	if item, exists := r.baseRegistry.Get(itemID); exists {
		// 更新现有项的验证器
		item.validator = validator
		item.updatedAt = time.Now()
		if !r.baseRegistry.Update(item) {
			return errors.NewError(errors.ErrCodeInternalErr, "failed to update adapter validator")
		}
	} else {
		// 创建仅包含验证器的项
		item := &AdapterItem{
			id:          "validator:" + adapterType,
			name:        "Validator for " + adapterType,
			adapterType: adapterType,
			validator:   validator,
			createdAt:   time.Now(),
			updatedAt:   time.Now(),
		}
		if err := r.baseRegistry.Register(item); err != nil {
			return errors.WrapError(errors.ErrCodeInternalErr, "failed to register adapter validator", err)
		}
	}

	return nil
}

// CreateAdapter 创建适配器实例
func (r *AdapterRegistry) CreateAdapter(name, adapterType string, config interface{}) (ModelAdapter, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查是否已存在同名适配器
	if _, exists := r.baseRegistry.Get(name); exists {
		return nil, NewInvalidParametersError(fmt.Sprintf("adapter with name '%s' already exists", name), nil)
	}

	// 获取适配器工厂
	factory, exists := r.factories[adapterType]
	if !exists {
		return nil, NewModelNotSupportedError(fmt.Sprintf("unsupported adapter type: %s", adapterType), nil)
	}

	// 验证配置
	if validator, exists := r.validators[adapterType]; exists {
		if err := validator(config); err != nil {
			return nil, NewInvalidConfigError(fmt.Sprintf("invalid config for adapter type '%s'", adapterType), err)
		}
	}

	// 创建适配器实例
	adapter, err := factory(config)
	if err != nil {
		return nil, NewClientCreationFailedError(fmt.Sprintf("failed to create adapter '%s'", name), err)
	}

	// 获取适配器信息
	factoryItemID := "factory:" + adapterType
	var info AdapterInfo
	if factoryItem, exists := r.baseRegistry.Get(factoryItemID); exists {
		info = factoryItem.info
	} else {
		// 如果找不到工厂项，使用默认信息
		info = AdapterInfo{
			Name: adapterType,
			Type: adapterType,
		}
	}

	// 创建适配器实例项并注册到基础注册表
	item := &AdapterItem{
		id:          name,
		name:        name,
		adapterType: adapterType,
		adapter:     adapter,
		info:        info,
		factory:     factory,
		validator:   r.validators[adapterType],
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
	}

	if err := r.baseRegistry.Register(item); err != nil {
		return nil, errors.WrapError(errors.ErrCodeInternalErr, "failed to register adapter instance", err)
	}

	return adapter, nil
}

// GetAdapter 获取适配器实例
func (r *AdapterRegistry) GetAdapter(name string) (ModelAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if item, exists := r.baseRegistry.Get(name); exists {
		return item.adapter, true
	}
	return nil, false
}

// RemoveAdapter 移除适配器实例
func (r *AdapterRegistry) RemoveAdapter(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.baseRegistry.Contains(name) {
		return NewClientNotFoundError(fmt.Sprintf("adapter not found: %s", name), nil)
	}

	// 简化版本：直接移除适配器，不进行额外清理
	// 适配器的资源清理由垃圾回收器处理
	if !r.baseRegistry.Remove(name) {
		return errors.NewError(errors.ErrCodeInternalErr, "failed to remove adapter")
	}

	return nil
}

// ListAdapters 列出所有已创建的适配器
func (r *AdapterRegistry) ListAdapters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := r.baseRegistry.List()
	var names []string
	for _, item := range items {
		// 只返回实例项，不返回工厂或验证器项
		if item.adapter != nil {
			names = append(names, item.name)
		}
	}

	return names
}

// ListSupportedTypes 列出所有支持的适配器类型
func (r *AdapterRegistry) ListSupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 使用旧映射中的类型（向后兼容）
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

	itemID := "factory:" + adapterType
	if item, exists := r.baseRegistry.Get(itemID); exists {
		return item.info, true
	}

	return AdapterInfo{}, false
}

// GetAllAdapterInfos 获取所有适配器类型信息
func (r *AdapterRegistry) GetAllAdapterInfos() map[string]AdapterInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make(map[string]AdapterInfo)
	items := r.baseRegistry.GetByType("factory")
	for _, item := range items {
		if item.info.Type != "" {
			infos[item.info.Type] = item.info
		}
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

	count := 0
	items := r.baseRegistry.List()
	for _, item := range items {
		// 只计算实例项，不计算工厂或验证器项
		if item.adapter != nil {
			count++
		}
	}

	return count
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

	// 清理所有适配器实例，但保留工厂和验证器
	items := r.baseRegistry.List()
	for _, item := range items {
		if item.adapter != nil {
			r.baseRegistry.Remove(item.id)
		}
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

	// 复制工厂和验证器项到新注册表
	items := r.baseRegistry.List()
	for _, item := range items {
		if item.adapter == nil { // 只复制工厂和验证器项，不复制适配器实例
			newItem := &AdapterItem{
				id:          item.id,
				name:        item.name,
				adapterType: item.adapterType,
				info:        item.info,
				factory:     item.factory,
				validator:   item.validator,
				createdAt:   item.createdAt,
				updatedAt:   time.Now(),
			}
			newRegistry.baseRegistry.Register(newItem)
		}
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
	var typeNames []string
	adapterCount := 0
	factoryCount := 0
	validatorCount := len(r.validators)

	items := r.baseRegistry.List()
	for _, item := range items {
		if item.adapter != nil {
			adapterNames = append(adapterNames, item.name)
			adapterCount++
		} else if item.factory != nil {
			factoryCount++
			typeNames = append(typeNames, item.adapterType)
		}
	}

	// 如果基础注册表没有数据，使用旧映射中的数据
	if factoryCount == 0 {
		factoryCount = len(r.factories)
		for typeName := range r.factories {
			typeNames = append(typeNames, typeName)
		}
	}

	return RegistryStats{
		SupportedTypes:       factoryCount,
		CreatedAdapters:      adapterCount,
		RegisteredValidators: validatorCount,
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
