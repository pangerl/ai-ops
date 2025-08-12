package util

import (
	"fmt"
	"sync"
)

// RegistryService 负责管理应用中所有类型的注册表实例
type RegistryService struct {
	mu         sync.RWMutex
	registries map[string]interface{}
}

var (
	globalRegistryService *RegistryService
	once                  sync.Once
)

// NewRegistryService 创建一个新的中央注册服务
func NewRegistryService() *RegistryService {
	return &RegistryService{
		registries: make(map[string]interface{}),
	}
}

// GetRegistryService 获取全局唯一的注册服务实例
func GetRegistryService() *RegistryService {
	once.Do(func() {
		globalRegistryService = NewRegistryService()
	})
	return globalRegistryService
}

// Register 新建并注册一个指定类型的注册表
// key 是该注册表的唯一标识符，例如 "llm_adapters" 或 "tools"
func (s *RegistryService) Register(key string, registryInstance interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.registries[key]; exists {
		return fmt.Errorf("registry with key '%s' already exists", key)
	}

	s.registries[key] = registryInstance
	return nil
}

// Get 根据键名获取一个注册表实例
// 返回的实例需要进行类型断言
func (s *RegistryService) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	instance, exists := s.registries[key]
	return instance, exists
}
