package llm

import (
	"ai-ops/internal/util"
	"ai-ops/internal/util/errors"
	"ai-ops/pkg/registry"
	"fmt"
	"sync"
	"time"
)

const (
	// LLMRegistryKey 是 LLM 适配器注册表在中央服务中的键名
	LLMRegistryKey = "llm_adapters"
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

// --- 全局注册表实例 ---
var (
	llmRegistry registry.Registry[*AdapterItem]
	llmMutex    sync.RWMutex
)

// InitRegistry 初始化LLM注册表
func InitRegistry() error {
	llmMutex.Lock()
	defer llmMutex.Unlock()
	
	if llmRegistry != nil {
		return nil // 已经初始化
	}
	regService := util.GetRegistryService()
	reg := registry.NewRegistry[*AdapterItem]()
	err := regService.Register(LLMRegistryKey, reg)
	if err != nil {
		// 如果注册失败（例如，键已存在），则尝试获取现有实例
		if instance, ok := regService.Get(LLMRegistryKey); ok {
			if registry, ok := instance.(registry.Registry[*AdapterItem]); ok {
				llmRegistry = registry
				return nil
			} else {
				return fmt.Errorf("LLM注册表类型断言失败")
			}
		} else {
			// 这是一个严重错误，表示注册服务状态不一致
			return fmt.Errorf("初始化或获取LLM注册表失败: %v", err)
		}
	} else {
		llmRegistry = reg
	}
	return nil
}

// getRegistry 获取LLM注册表实例
func getRegistry() (registry.Registry[*AdapterItem], error) {
	llmMutex.RLock()
	defer llmMutex.RUnlock()
	
	if llmRegistry == nil {
		return nil, errors.NewError(errors.ErrCodeInternalErr, "LLM registry not initialized")
	}
	return llmRegistry, nil
}

// RegisterAdapterFactory 注册适配器工厂函数
func RegisterAdapterFactory(adapterType string, factory AdapterFactory, info AdapterInfo) error {
	reg, err := getRegistry()
	if err != nil {
		return err
	}

	if adapterType == "" {
		return errors.NewError(errors.ErrCodeInvalidParameters, "adapter type cannot be empty")
	}
	if factory == nil {
		return errors.NewError(errors.ErrCodeInvalidParameters, "adapter factory cannot be nil")
	}

	item := &AdapterItem{
		id:          "factory:" + adapterType,
		name:        info.Name,
		adapterType: adapterType,
		factory:     factory,
		info:        info,
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
	}

	return reg.Register(item)
}

// RegisterConfigValidator 注册配置验证器
func RegisterConfigValidator(adapterType string, validator ConfigValidator) error {
	reg, err := getRegistry()
	if err != nil {
		return err
	}

	if adapterType == "" {
		return errors.NewError(errors.ErrCodeInvalidParameters, "adapter type cannot be empty")
	}
	if validator == nil {
		return errors.NewError(errors.ErrCodeInvalidParameters, "config validator cannot be nil")
	}

	itemID := "factory:" + adapterType
	if item, exists := reg.Get(itemID); exists {
		item.validator = validator
		item.updatedAt = time.Now()
		if !reg.Update(item) {
			return errors.NewError(errors.ErrCodeInternalErr, "更新适配器验证器失败")
		}
	} else {
		// 如果工厂不存在，也允许注册验证器，以便后续使用
		item := &AdapterItem{
			id:          "validator:" + adapterType,
			name:        "Validator for " + adapterType,
			adapterType: adapterType,
			validator:   validator,
			createdAt:   time.Now(),
			updatedAt:   time.Now(),
		}
		return reg.Register(item)
	}
	return nil
}

// CreateAdapter 创建适配器实例
func CreateAdapter(name, adapterType string, config interface{}) (ModelAdapter, error) {
	reg, err := getRegistry()
	if err != nil {
		return nil, err
	}

	if _, exists := reg.Get(name); exists {
		return nil, errors.NewError(errors.ErrCodeInvalidParameters, fmt.Sprintf("adapter with name '%s' already exists", name))
	}

	factoryItemID := "factory:" + adapterType
	factoryItem, exists := reg.Get(factoryItemID)
	if !exists || factoryItem.factory == nil {
		return nil, errors.NewError(errors.ErrCodeModelNotSupported, fmt.Sprintf("unsupported adapter type: %s", adapterType))
	}

	if factoryItem.validator != nil {
		if err := factoryItem.validator(config); err != nil {
			return nil, errors.WrapError(errors.ErrCodeInvalidConfig, fmt.Sprintf("invalid config for adapter type '%s'", adapterType), err)
		}
	}

	adapter, err := factoryItem.factory(config)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeClientCreationFailed, fmt.Sprintf("failed to create adapter '%s'", name), err)
	}

	item := &AdapterItem{
		id:          name,
		name:        name,
		adapterType: adapterType,
		adapter:     adapter,
		info:        factoryItem.info,
		factory:     factoryItem.factory,
		validator:   factoryItem.validator,
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
	}

	if err := reg.Register(item); err != nil {
		return nil, errors.WrapError(errors.ErrCodeInternalErr, fmt.Sprintf("注册适配器实例失败，名称: %s", name), err)
	}

	return adapter, nil
}

// GetAdapter 获取适配器实例
func GetAdapter(name string) (ModelAdapter, bool) {
	reg, err := getRegistry()
	if err != nil {
		return nil, false
	}

	if item, exists := reg.Get(name); exists && item.adapter != nil {
		return item.adapter, true
	}
	return nil, false
}

// RemoveAdapter 移除适配器实例
func RemoveAdapter(name string) error {
	reg, err := getRegistry()
	if err != nil {
		return err
	}
	if !reg.Remove(name) {
		return errors.NewError(errors.ErrCodeClientNotFound, fmt.Sprintf("未找到适配器: %s", name))
	}
	return nil
}

// ListAdapters 列出所有已创建的适配器
func ListAdapters() []string {
	reg, err := getRegistry()
	if err != nil {
		return nil
	}

	items := reg.List()
	var names []string
	for _, item := range items {
		if item.adapter != nil {
			names = append(names, item.name)
		}
	}
	return names
}

// ListSupportedTypes 列出所有支持的适配器类型
func ListSupportedTypes() []string {
	reg, err := getRegistry()
	if err != nil {
		return nil
	}

	items := reg.GetByType("factory")
	var types []string
	for _, item := range items {
		types = append(types, item.adapterType)
	}
	return types
}

// GetAdapterInfo 获取适配器类型信息
func GetAdapterInfo(adapterType string) (AdapterInfo, bool) {
	reg, err := getRegistry()
	if err != nil {
		return AdapterInfo{}, false
	}

	itemID := "factory:" + adapterType
	if item, exists := reg.Get(itemID); exists {
		return item.info, true
	}
	return AdapterInfo{}, false
}

// GetAllAdapterInfos 获取所有适配器类型信息
func GetAllAdapterInfos() map[string]AdapterInfo {
	reg, err := getRegistry()
	if err != nil {
		return nil
	}

	infos := make(map[string]AdapterInfo)
	items := reg.List()
	for _, item := range items {
		// 确保是工厂定义项
		if item.factory != nil && item.info.Type != "" {
			infos[item.info.Type] = item.info
		}
	}
	return infos
}
