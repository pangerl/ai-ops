# 插件工具目录

这个目录包含所有的插件工具实现。每个工具都是一个独立的Go文件。

## 目录结构

```
plugins/
├── README.md          # 本文档
├── init.go           # 插件包初始化
├── echo_tool.go      # 回显工具
└── weather_tool.go   # 天气工具
```

## 如何添加新工具

1. **创建工具文件**：在此目录下创建 `your_tool.go` 文件

2. **实现Tool接口**：
```go
package plugins

import (
    "context"
    "ai-ops/internal/util"
)

type YourTool struct{}

func (t *YourTool) Name() string {
    return "your_tool"
}

func (t *YourTool) Description() string {
    return "你的工具描述"
}

func (t *YourTool) Parameters() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "param1": map[string]any{
                "type": "string",
                "description": "参数描述",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *YourTool) Execute(ctx context.Context, args map[string]any) (string, error) {
    // 实现你的工具逻辑
    return "result", nil
}

func NewYourTool() interface{} {
    return &YourTool{}
}
```

3. **更新plugin_loader.go**：在 `LoadPlugins` 方法中添加你的工具：
```go
tools := []Tool{
    plugins.NewEchoTool().(*plugins.EchoTool),
    plugins.NewWeatherTool().(*plugins.WeatherTool),
    plugins.NewYourTool().(*plugins.YourTool), // 添加这行
}
```

## 现有工具

### EchoTool (echo_tool.go)
- **功能**：回显输入的消息
- **参数**：`message` (string) - 要回显的消息内容
- **用法**：用于测试工具调用功能

### WeatherTool (weather_tool.go)
- **功能**：查询指定地点的实时天气信息
- **参数**：`location` (string) - 城市名称、LocationID或经纬度坐标
- **配置**：需要在config.toml中配置和风天气API密钥
- **用法**：提供实时天气查询服务

## 工具开发最佳实践

1. **错误处理**：使用 `util.NewError()` 创建标准化错误
2. **日志记录**：使用 `util.Infow()` 等函数记录执行日志
3. **参数验证**：严格验证输入参数的类型和有效性
4. **上下文处理**：正确处理 `context.Context` 用于超时和取消
5. **配置管理**：通过 `config.Config` 访问应用配置
6. **资源清理**：确保正确关闭HTTP连接等资源

## 自动加载机制

当应用启动时，`plugin_loader.go` 会自动：
1. 扫描plugins目录
2. 创建工具实例
3. 注册到工具管理器
4. 记录加载结果

工具加载完成后，就可以通过AI对话调用这些工具了。