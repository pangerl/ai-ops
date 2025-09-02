# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Building
```bash
go build -o ai-ops .
```

### Running
```bash
# 启动交互式对话模式
./ai-ops chat

# 启用智能体模式（自主规划和执行任务）
./ai-ops chat -a

# 智能体模式 + 显示思考过程
./ai-ops chat -a -t

# 显示帮助
./ai-ops --help

# 显示版本信息
./ai-ops version

# 配置管理
./ai-ops config

# MCP服务管理
./ai-ops mcp
```

### Development
```bash
# 格式化代码
gofmt -w .

# 静态分析
go vet ./...

# 整理依赖
go mod tidy

# 运行测试（目前无测试文件）
go test ./...
```

## Architecture

这是一个基于 Go 的智能运维命令行工具，采用模块化设计，聚合大语言模型能力与自动化运维场景。

### 核心架构组件

1. **LLM 适配器系统**（`internal/llm/`）
   - 统一的模型适配器接口，支持多个 LLM 提供商
   - 注册表模式管理适配器实例
   - 支持 OpenAI、Gemini、GLM 等模型
   - 配置驱动的客户端创建和管理

2. **工具系统**（`internal/tools/`）
   - 插件化的工具架构，AI 可在对话中自动调用
   - 工具管理器统一管理所有工具实例
   - 插件工厂模式支持动态注册
   - 内置工具：echo、weather、RAG、sysinfo

3. **MCP 集成**（`internal/mcp/`）
   - 支持 Model Context Protocol 规范
   - MCP 服务管理和工具桥接
   - 外部工具服务集成能力

4. **注册表系统**（`pkg/registry/`）
   - 通用的注册表框架，支持泛型
   - 中央注册服务统一管理各种组件
   - 工厂模式和服务发现

### 目录结构

- `cmd/` - CLI 命令定义和应用初始化
- `internal/chat/` - 交互式对话界面（TUI）+ 智能体模式
- `internal/config/` - 配置管理
- `internal/llm/` - LLM 适配器和注册表
- `internal/mcp/` - MCP 协议支持
- `internal/tools/` - 工具系统和插件
- `internal/util/` - 通用工具和错误处理
- `pkg/registry/` - 通用注册表框架

### 配置

主配置文件为 `config.toml`，支持：
- 多个 AI 模型配置（API密钥通过环境变量）
- 日志配置
- 各工具的专用配置（天气API、RAG服务等）

本地配置可用 `local-config.toml` 覆盖默认配置。

### 扩展开发

新工具插件开发：
1. 在 `internal/tools/plugins/` 创建工具实现
2. 实现 `Tool` 接口
3. 在 `init.go` 中注册工厂函数
4. AI 会自动发现并调用新工具

### 依赖

- Go 1.24.5+
- Cobra（CLI框架）
- Model Context Protocol SDK
- 其他UI和网络库