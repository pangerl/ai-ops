// Package registry 提供简化的注册表接口和实现
//
// 这个包实现了一个类型安全的泛型注册表系统，支持：
// - 类型安全的注册、获取、列表、移除和清空操作
// - 按类型分类管理注册项
// - 线程安全的并发访问
//
// 基本用法：
//
//  1. 定义注册表项：
//     type MyItem struct {
//     id   string
//     name string
//     typ  string
//     data interface{}
//     }
//
//     // 实现 RegistryItem 接口
//     func (i *MyItem) ID() string { return i.id }
//     func (i *MyItem) Name() string { return i.name }
//     func (i *MyItem) Type() string { return i.typ }
//
//  2. 创建注册表：
//     reg := registry.NewRegistry[*MyItem]()
//
//  3. 注册项目：
//     item := NewMyItem("item1", "示例项目", "typeA", "数据")
//     err := reg.Register(item)
//
//  4. 获取项目：
//     if item, exists := reg.Get("item1"); exists {
//     // 使用项目
//     }
//
//  5. 列出所有项目：
//     items := reg.List()
//     for _, item := range items {
//     fmt.Printf("%s: %v\n", item.Name(), item.Data())
//     }
//
//  6. 按类型获取项目：
//     typeAItems := reg.GetByType("typeA")
//
//  7. 更新和移除项目：
//     updated := reg.Update(updatedItem)
//     removed := reg.Remove("item1")
//
//  8. 检查项目是否存在：
//     exists := reg.Contains("item1")
//
//  9. 清空注册表：
//     reg.Clear()
package registry
