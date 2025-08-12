package registry

import (
	"sync"
)

// BaseRegistry 是注册表的基础实现
type BaseRegistry[T RegistryItem] struct {
	mu    sync.RWMutex
	items map[string]T
}

// NewBaseRegistry 创建一个新的基础注册表实例
func NewBaseRegistry[T RegistryItem]() *BaseRegistry[T] {
	return &BaseRegistry[T]{
		items: make(map[string]T),
	}
}

// Register 注册一个新的项目到注册表
func (r *BaseRegistry[T]) Register(item T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := item.ID()
	r.items[id] = item
	return nil
}

// Get 根据ID从注册表中获取项目
func (r *BaseRegistry[T]) Get(id string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.items[id]
	return item, exists
}

// List 列出注册表中的所有项目
func (r *BaseRegistry[T]) List() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]T, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items
}

// Remove 从注册表中移除指定ID的项目
func (r *BaseRegistry[T]) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.items[id]
	if !exists {
		return false
	}

	delete(r.items, id)
	return true
}

// Clear 清空注册表中的所有项目
func (r *BaseRegistry[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.items = make(map[string]T)
}

// GetByType 根据类型获取所有项目
func (r *BaseRegistry[T]) GetByType(itemType string) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []T
	for _, item := range r.items {
		if item.Type() == itemType {
			items = append(items, item)
		}
	}
	return items
}

// Contains 检查注册表中是否存在指定ID的项目
func (r *BaseRegistry[T]) Contains(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.items[id]
	return exists
}

// Update 更新注册表中的项目
func (r *BaseRegistry[T]) Update(item T) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := item.ID()
	if _, exists := r.items[id]; !exists {
		return false
	}

	r.items[id] = item
	return true
}
