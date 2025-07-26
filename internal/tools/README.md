# 工具模块 (internal/tools)

## 1. 模块概述

`tools` 模块负责系统中所有工具（Tools）的定义、注册、管理和执行。它提供了一个可扩展的插件化框架，允许动态地添加和使用新工具，同时为工具的执行提供了超时和重试等增强功能。

## 2. 核心组件

-   **`types.go`**: 定义了模块的核心接口和数据结构。
    -   `Tool`: 所有工具必须实现的接口，包含 `Name`, `Description`, `Parameters`, `Execute` 方法。
    -   `ToolDefinition`: 用于向 AI 模型描述工具的结构体。
    -   `ToolCall`: 代表一次具体的工具调用请求。

-   **`registry.go`**: 提供了 `ToolRegistry`，一个线程安全的底层工具注册表，负责工具的存储和按名称检索。

-   **`manager.go`**: 提供了 `ToolManager`，它是对 `ToolRegistry` 的封装，提供了更高级的工具管理和执行流程，包括日志记录和错误处理。

-   **`plugin_registry.go`**: 实现了基于**工厂模式**的插件自动注册机制。这是实现工具插件化的核心，允许工具“自我注册”。

-   **`executor.go`**: 提供了 `ToolCallExecutor`，它包装了 `ToolManager`，为工具执行增加了**超时控制**和**自动重试**等高级功能。

## 3. 工具实现与注册方法

本模块采用基于**工厂模式的插件化架构**，实现了高度解耦和可扩展性。这是系统中添加和管理工具的唯一推荐方法。

### 如何添加一个新工具？

1.  **创建工具文件**: 在 `internal/tools/plugins/` 目录下为你的新工具创建一个 Go 文件（例如 `my_tool.go`）。

2.  **实现 `Tool` 接口**: 在新文件中，创建一个结构体并实现 `tools.Tool` 接口的所有方法 (`Name`, `Description`, `Parameters`, `Execute`)。

3.  **创建工厂函数**: 定义一个函数，它返回你的工具结构体的新实例。这个函数必须符合 `PluginFactory` 类型 (`func() interface{}`)。

4.  **使用 `init()` 自动注册**: 在文件的 `init()` 函数中，调用 `tools.RegisterPluginFactory`，将你的工具名称和工厂函数注册到全局插件注册表中。

### 示例: `my_tool.go`

```go
package plugins

import (
    "context"
    "ai-ops/internal/tools"
)

// 1. 定义工具结构体
type MyTool struct{}

// 确保结构体实现了 Tool 接口
var _ tools.Tool = (*MyTool)(nil)

// 2. 实现接口方法
func (t *MyTool) Name() string { return "my_tool" }
func (t *MyTool) Description() string { return "这是一个新工具的描述" }
func (t *MyTool) Parameters() map[string]any { /* ... 定义参数 ... */ return nil }
func (t *MyTool) Execute(ctx context.Context, args map[string]any) (string, error) {
    // ... 执行工具逻辑 ...
    return "执行成功", nil
}

// 3. 定义工厂函数
func NewMyTool() interface{} {
    return &MyTool{}
}

// 4. 在 init 中自动注册
func init() {
    tools.RegisterPluginFactory("my_tool", NewMyTool)
}
```

## 4. 应用程序初始化流程

在应用程序启动时，应遵循以下步骤来初始化并使用工具系统：

```go
// 1. 创建一个工具管理器
toolManager := tools.NewToolManager()

// 2. 从插件注册表中创建所有自动注册的工具
//    这一步会自动找到所有通过 init() 注册的插件
pluginTools := tools.CreatePluginTools()

// 3. 将插件工具注册到管理器
for _, tool := range pluginTools {
    if err := toolManager.RegisterTool(tool); err != nil {
        // 建议记录错误日志
    }
}

// 4. 创建带增强功能的执行器
toolExecutor := tools.NewToolCallExecutor(toolManager)
toolExecutor.SetRetryConfig(3, 1000) // (可选) 设置重试次数和延迟

// 5. 在业务逻辑中使用 toolExecutor 来执行工具调用
// result, err := toolExecutor.ExecuteWithRetryAndTimeout(ctx, toolCall, 5000)
```

## 5. 最终目录结构

重构后的目录结构更加简洁和聚焦：

```
internal/tools/
├── manager.go           # 核心管理器
├── registry.go          # 底层注册表
├── plugin_registry.go   # 插件化核心
├── executor.go          # 工具执行器 (超时/重试)
├── types.go             # 核心类型
└── plugins/             # 存放所有工具插件
    ├── echo_tool.go
    ├── weather_tool.go
    └── ... (其他工具)
```
