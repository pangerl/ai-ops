package registry

// RegistryItem 定义注册表项的基本接口
type RegistryItem interface {
	// ID 返回注册表项的唯一标识符
	ID() string
	// Name 返回注册表项的名称
	Name() string
	// Type 返回注册表项的类型
	Type() string
}

// Registry 定义泛型注册表接口
type Registry[T RegistryItem] interface {
	// Register 注册一个新的项目到注册表
	Register(item T) error
	// Get 根据ID从注册表中获取项目
	Get(id string) (T, bool)
	// List 列出注册表中的所有项目
	List() []T
	// Remove 从注册表中移除指定ID的项目
	Remove(id string) bool
	// Clear 清空注册表中的所有项目
	Clear()
	// GetByType 根据类型获取所有项目
	GetByType(itemType string) []T
	// Contains 检查注册表中是否存在指定ID的项目
	Contains(id string) bool
	// Update 更新注册表中的项目
	Update(item T) bool
}
