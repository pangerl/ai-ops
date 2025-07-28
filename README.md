# AI-Ops

AI-Ops 是一个基于 Go 开发的智能运维命令行工具，旨在将大型语言模型（LLM）的能力与自动化运维任务相结合。它提供了一个灵活的框架，可以通过插件扩展其功能，实现与 AI 的交互式对话、调用外部工具和执行自动化运维脚本。

## ✨ 功能特性

- **交互式对话**：通过 `chat` 命令，可以与配置的 AI 模型（如 OpenAI GPT、Google Gemini）进行交互式对话。
- **强大的工具系统**：支持通过插件扩展工具集，AI 可以在对话中根据上下文智能调用这些工具来完成特定任务（例如查询天气、获取监控信息等）。
- **多模型支持**：支持同时配置多个 AI 大模型，并可以在对话中轻松切换，适配 Openai、Gemini。
- **灵活的配置**：通过一个简单的 TOML 文件进行所有配置，包括 AI 模型、API 密钥、日志等。
- **可扩展性**：基于 Go 语言开发，性能优异，且易于二次开发和功能扩展。

## 🚀 快速开始

### 1. 先决条件

- [Go](https://golang.org/dl/) (1.21 或更高版本)
- Git

### 2. 克隆与编译

```bash
# 克隆项目
git clone https://github.com/pangerl/ai-ops.git
cd ai-ops

# 编译项目
go build -o ai-ops .
```

### 3. 创建配置文件

在项目根目录创建一个 `config.toml` 文件，并填入您的配置。

```toml
# config.toml 示例

# AI 相关配置
[ai]
default_model = "gemini"  # 默认使用的AI模型
timeout = 30              # AI请求超时时间（秒）

# 配置多个AI模型
[ai.models.gemini]
type = "gemini"
api_key = "${GEMINI_API_KEY}" # 支持从环境变量读取
model = "gemini-1.5-flash"

[ai.models.openai]
type = "openai"
api_key = "${OPENAI_API_KEY}"
model = "gpt-4o-mini"

# 日志配置
[logging]
level = "info"      # 日志级别 (debug, info, warn, error)
format = "text"     # 日志格式 (text, json)
output = "stdout"   # 日志输出 (stdout, stderr, file)
file = ""           # 如果 output 设置为 file，则需要指定日志文件路径

# 天气工具的特定配置
[weather]
api_host = "https://devapi.qweather.com"
api_key = "${QWEATHER_API_KEY}" # 和风天气API Key
```

### 4. 运行

完成编译和配置后，您可以开始使用 `ai-ops`。

```bash
# 查看帮助信息
./ai-ops --help

# 启动交互式对话
./ai-ops chat
```

## 📖 使用示例

启动交互式对话后，您可以直接与 AI 对话，或者让它调用工具。

```
> ./ai-ops chat
AI-Ops 框架初始化完成
配置文件加载成功
默认AI模型: gpt-4-turbo (openai)
日志级别: info

正在启动交互式对话模式...
使用默认模型: openai
你好！有什么可以帮助你的吗？
> 今天北京天气怎么样？

AI 正在思考...
AI 正在调用工具: weather({"location":"北京"})
AI 正在处理工具返回结果...

根据最新的天气数据，北京市当前天气为晴，气温25℃，体感温度26℃，微风。
```

## 🔧 如何开发

### 添加新工具

1.  在 `internal/tools/plugins/` 目录下创建一个新的 Go 文件，例如 `my_tool.go`。
2.  实现 `tools.Tool` 接口（`Name`, `Description`, `Parameters`, `Execute`）。
3.  在 `internal/tools/plugins/init.go` 文件中的 `init()` 函数里注册你的新工具。

可以参考 `internal/tools/plugins/echo_tool.go` 作为实现示例。
更多工具详细请点击：[工具](docs/tools.md)

### 项目结构

```
.
├── cmd/                # Cobra 命令定义
├── internal/           # 内部业务逻辑
│   ├── ai/             # AI 客户端（OpenAI, Gemini）
│   ├── chat/           # 交互式对话逻辑
│   ├── config/         # 配置加载
│   ├── tools/          # 工具管理和插件系统
│   └── util/           # 通用工具函数
├── main.go             # 程序入口
└── config.toml         # 配置文件
```

## 📄 开源许可

本项目基于 [MIT License](LICENSE) 开源。
