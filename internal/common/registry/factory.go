package registry

// NewRegistry 创建一个新的注册表
func NewRegistry[T RegistryItem]() Registry[T] {
	return NewBaseRegistry[T]()
}
