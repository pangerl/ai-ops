package registry

import (
	"fmt"
)

// ExampleItem 示例注册表项实现
type ExampleItem struct {
	id   string
	name string
	typ  string
	data interface{}
}

// NewExampleItem 创建一个新的示例项
func NewExampleItem(id, name, itemType string, data interface{}) *ExampleItem {
	return &ExampleItem{
		id:   id,
		name: name,
		typ:  itemType,
		data: data,
	}
}

// ID 返回示例项的唯一标识符
func (i *ExampleItem) ID() string {
	return i.id
}

// Name 返回示例项的名称
func (i *ExampleItem) Name() string {
	return i.name
}

// Type 返回示例项的类型
func (i *ExampleItem) Type() string {
	return i.typ
}

// Data 返回示例项的数据
func (i *ExampleItem) Data() interface{} {
	return i.data
}

// ExampleUsage 展示注册表使用示例
func ExampleUsage() {
	// 创建一个新的注册表
	reg := NewBaseRegistry[*ExampleItem]()

	// 创建一些示例项
	item1 := NewExampleItem("item1", "示例项目1", "typeA", "数据1")
	item2 := NewExampleItem("item2", "示例项目2", "typeB", "数据2")
	item3 := NewExampleItem("item3", "示例项目3", "typeA", "数据3")

	// 注册项目
	reg.Register(item1)
	reg.Register(item2)
	reg.Register(item3)

	// 获取项目
	if item, exists := reg.Get("item1"); exists {
		fmt.Printf("找到项目: %s, 类型: %s\n", item.Name(), item.Type())
	}

	// 列出所有项目
	fmt.Println("\n所有项目:")
	for _, item := range reg.List() {
		fmt.Printf("- %s (%s): %v\n", item.Name(), item.Type(), item.Data())
	}

	// 按类型获取项目
	fmt.Println("\n类型为 typeA 的项目:")
	for _, item := range reg.GetByType("typeA") {
		fmt.Printf("- %s: %v\n", item.Name(), item.Data())
	}

	// 检查项目是否存在
	fmt.Printf("\n项目 item2 是否存在: %v\n", reg.Contains("item2"))

	// 更新项目
	updatedItem := NewExampleItem("item2", "更新的项目2", "typeB", "更新后的数据")
	if reg.Update(updatedItem) {
		fmt.Println("项目更新成功")
	}

	// 获取更新后的项目
	if item, exists := reg.Get("item2"); exists {
		fmt.Printf("更新后的项目: %s, 数据: %v\n", item.Name(), item.Data())
	}
}
