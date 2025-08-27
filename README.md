
# AI-Ops

AI-Ops 是一个基于 Go 的智能运维命令行工具，聚合大语言模型（LLM）能力与自动化运维场景，助力高效、智能的运维对话与工具调用。

## ✨ 核心特性

- **智能对话**：通过命令行与 AI 进行自然语言交互，支持多轮上下文对话
- **工具自动调用**：AI 可根据对话内容自动调用插件工具，完成系统监控、天气查询、知识检索等任务
- **多模型支持**：支持 OpenAI、Gemini、GLM 等主流大语言模型，可灵活切换
- **模块化架构**：采用注册表模式的模块化设计，支持插件扩展和 MCP 协议集成
- **系统监控**：内置系统信息工具，可实时监控 CPU、内存、磁盘、网络等状态

## 🏗️ 架构设计

### 核心组件

1. **LLM 适配器系统** (`internal/llm/`)
   - 统一的模型适配器接口，支持多个 LLM 提供商
   - 注册表模式管理适配器实例
   - 配置驱动的客户端创建和管理

2. **工具系统** (`internal/tools/`)
   - 插件化的工具架构，AI 可在对话中自动调用
   - 工具管理器统一管理所有工具实例
   - 插件工厂模式支持动态注册

3. **MCP 集成** (`internal/mcp/`)
   - 支持 Model Context Protocol 规范
   - MCP 服务管理和工具桥接
   - 外部工具服务集成能力

4. **注册表系统** (`pkg/registry/`)
   - 通用的注册表框架，支持泛型
   - 中央注册服务统一管理各种组件
   - 工厂模式和服务发现

## 🛠️ 内置工具

AI-Ops 内置了多种实用工具，AI 可在对话中自动调用：

### 系统监控工具 (sysinfo)
- **CPU 监控**：实时 CPU 使用率、核心信息、各核心负载情况
- **内存监控**：内存使用率、缓存、交换分区状态
- **磁盘监控**：磁盘分区使用情况、文件系统信息、Inode 统计
- **网络监控**：网络接口状态、IP 地址、流量统计
- **系统负载**：1/5/15 分钟负载平均值、主机信息、运行时间
- **系统概览**：一键查看所有关键系统指标

### 其他工具
- **weather**：天气查询工具，支持自然语言查询如"北京天气怎么样"
- **rag**：知识检索增强工具，可用于文档问答和知识库查询
- **echo**：回显测试工具，主要用于插件开发调试

### 工具扩展
- 所有工具均以插件形式注册，开发者可在 `internal/tools/plugins/` 目录下添加新工具
- 只需实现标准接口并注册，AI 即可在对话中自动发现和调用
- 参考 `weather_tool.go` 或 `sysinfo_tool.go` 示例进行开发

## 🎬 功能演示

向 AI 询问问题，并自动调用工具查询天气、监控系统状态、召回知识库数据。

![AI-Ops 功能演示](.github/readme/ai_ops.gif)

---

## 🚀 快速开始

### 安装

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd ai-ops
   ```

2. **编译程序**
   ```bash
   go build -o ai-ops .
   ```

3. **配置环境变量**
   ```bash
   # 根据使用的模型设置对应 API Key
   export GEMINI_API_KEY="your_gemini_api_key"
   export OPENAI_API_KEY="your_openai_api_key"
   export GLM_API_KEY="your_glm_api_key"
   
   # 天气工具 API Key（可选）
   export QWEATHER_API_KEY="your_qweather_api_key"
   ```

### 使用

1. **启动交互式对话**
   ```bash
   ./ai-ops chat
   ```

2. **与 AI 交互示例**
   ```
   > 查看当前系统 CPU 和内存使用情况
   AI 正在调用工具: sysinfo({"action":"overview"})
   
   📊 系统概览
   🔥 CPU: 15.2%
   💾 内存: 8.1 GB / 16.0 GB (50.6%)
   ⚡ 负载: 1.23, 1.45, 1.67
   🖥️  主机: MacBook-Pro (darwin)
   ⏰ 运行时间: 2天15小时30分钟
   ```

   ```
   > 今天北京天气怎么样？
   AI 正在调用工具: weather({"location":"北京"})
   
   北京市当前天气为晴，气温25℃，体感温度26℃，微风。
   ```

3. **其他命令**
   ```bash
   # 显示帮助
   ./ai-ops --help
   
   # 显示版本信息
   ./ai-ops version
   
   # 配置管理
   ./ai-ops config
   
   # MCP 服务管理
   ./ai-ops mcp
   ```

4. **退出对话**
   输入 `exit` 或 `quit` 即可安全退出。

## ⚙️ 配置说明

### 主配置文件 (config.toml)

```toml
[ai]
default_model = "gemini"  # 默认使用的模型
timeout = 30              # 请求超时时间(秒)

# AI 模型配置
[ai.models.gemini]
type = "gemini"
api_key = "${GEMINI_API_KEY}"
base_url = "https://generativelanguage.googleapis.com/v1beta"
model = "gemini-2.5-flash"

[ai.models.openai]
type = "openai"
api_key = "${OPENAI_API_KEY}"
base_url = "https://api.openai.com/v1/chat/completions"
model = "gpt-4o-mini"

# 工具配置
[weather]
api_host = "https://devapi.qweather.com"
api_key = "${QWEATHER_API_KEY}"

[rag]
enable = false  # 是否启用 RAG 工具
api_host = "http://localhost:8000"
retrieval_k = 15
top_k = 5

# 日志配置
[logging]
level = "warn"
format = "text"
output = "stdout"
```

### 本地配置覆盖

可创建 `local-config.toml` 文件覆盖默认配置，该文件会被 Git 忽略。

## 🗂️ 项目结构

```
ai-ops/
├── cmd/                    # CLI 命令定义
│   ├── chat.go            # 交互式对话命令
│   ├── config.go          # 配置管理命令
│   ├── mcp.go             # MCP 服务命令
│   └── ...
├── internal/
│   ├── chat/              # 交互式界面（TUI）
│   ├── config/            # 配置管理
│   ├── llm/               # LLM 适配器系统
│   │   ├── adapter.go     # 适配器接口
│   │   ├── registry.go    # 注册表
│   │   ├── openai.go      # OpenAI 适配器
│   │   ├── gemini.go      # Gemini 适配器
│   │   └── ...
│   ├── mcp/               # MCP 协议支持
│   ├── tools/             # 工具系统
│   │   ├── manager.go     # 工具管理器
│   │   └── plugins/       # 内置工具插件
│   │       ├── sysinfo_tool.go    # 系统监控工具
│   │       ├── weather_tool.go    # 天气查询工具
│   │       ├── rag_tool.go        # RAG 工具
│   │       └── echo_tool.go       # 调试工具
│   └── util/              # 通用工具
├── pkg/
│   └── registry/          # 通用注册表框架
├── config.toml            # 主配置文件
├── local-config.toml      # 本地配置（可选）
└── main.go               # 程序入口
```

## 🔧 扩展开发

### 新增工具插件

1. **实现工具接口**
   在 `internal/tools/plugins/` 目录下创建新工具文件：

   ```go
   package plugins
   
   import "context"
   
   type MyTool struct{}
   
   func NewMyTool() interface{} {
       return &MyTool{}
   }
   
   func (t *MyTool) ID() string {
       return "mytool"
   }
   
   func (t *MyTool) Name() string {
       return "我的工具"
   }
   
   func (t *MyTool) Type() string {
       return "custom"
   }
   
   func (t *MyTool) Description() string {
       return "工具功能描述"
   }
   
   func (t *MyTool) Parameters() map[string]any {
       return map[string]any{
           "type": "object",
           "properties": map[string]any{
               "param": map[string]any{
                   "type": "string",
                   "description": "参数说明",
               },
           },
           "required": []string{"param"},
       }
   }
   
   func (t *MyTool) Execute(ctx context.Context, args map[string]any) (string, error) {
       // 实现工具逻辑
       return "执行结果", nil
   }
   ```

2. **注册工具**
   在 `internal/tools/plugins/init.go` 中注册新工具：

   ```go
   func init() {
       registry.RegisterFactory("mytool", NewMyTool)
   }
   ```

3. **测试工具**
   重新编译程序，启动对话即可使用新工具。

### 新增 LLM 适配器

1. **实现适配器接口**
   在 `internal/llm/` 目录下创建新适配器文件，实现 `LLMAdapter` 接口

2. **注册适配器**
   在适配器包的 `init()` 函数中注册

3. **配置模型**
   在 `config.toml` 中添加新模型配置

## 🤝 贡献指南

1. Fork 本项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 📝 版本历史

- **v1.0.0** - 初始版本，包含基础对话和工具调用功能
- **v1.1.0** - 新增系统监控工具 (sysinfo)
- **v1.2.0** - 新增 MCP 协议支持

## 📄 开源许可

本项目基于 [MIT License](LICENSE) 开源。
