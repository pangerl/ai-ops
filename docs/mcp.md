# MCP 命令使用文档

`ai-ops mcp` 命令用于管理模型上下文协议 (Model Context Protocol, MCP) 的相关服务。通过此命令，您可以查看服务状态、管理工具以及与 MCP 服务器进行交互。

## 1. `mcp status`

显示 MCP 服务的当前状态。

### 功能

此命令会检查 `mcp_settings.json` 配置文件，并报告以下信息：
- 已配置的服务器总数。
- 成功连接的服务器数量。
- 每个服务器的连接状态（已连接或未连接）。
- 通过 MCP 注册的工具总数。

### 用法

```bash
ai-ops mcp status
```

### 示例输出

```
MCP服务状态:
============
配置文件: mcp_settings.json
已配置服务器数量: 2
已连接服务器数量: 1

服务器状态:
  fetch: ✅ 已连接
  google: ❌ 未连接

已注册的MCP工具数量: 5
```

## 2. `mcp list`

列出所有可用的 MCP 工具。

### 功能

此命令会显示从所有已连接的 MCP 服务器加载的工具列表。它会按服务器对工具进行分组，并提供每个工具的名称和描述。

### 用法

```bash
ai-ops mcp list
```

### 示例输出

```
可用的MCP工具:
==============

服务器: fetch
--------
  • fetch
    Fetches a URL from the internet and optionally extracts its contents as markdown.

其他工具:
--------
  • echo - A simple tool that echoes back the input.

总计: 2 个工具 (其中 1 个MCP工具)
```

## 3. `mcp test`

测试与 `mcp_settings.json` 中配置的所有 MCP 服务器的连接。

### 功能

此命令会尝试初始化 MCP 服务并连接到所有已配置的服务器。它会明确显示每个服务器的连接测试结果，并在出现问题时提供故障排除建议。

### 用法

```bash
ai-ops mcp test
```

### 示例输出

```
测试MCP服务器连接:
==================
正在初始化MCP服务...
✅ MCP服务初始化成功

连接结果:
  ✅ fetch: 连接成功
  ❌ google: 连接失败

🎉 成功连接 1 个服务器: [fetch]
```

## 4. `mcp call`

调用一个指定的 MCP 工具并执行它。

### 功能

此命令允许您直接从命令行执行任何已注册的 MCP 工具。您需要提供工具的名称以及一个 JSON 格式的字符串作为参数。

### 用法

```bash
ai-ops mcp call [tool_name] [arguments_json]
```

- `[tool_name]` (必需): 要调用的工具的名称。
- `[arguments_json]` (可选): 一个有效的 JSON 字符串，包含工具所需的参数。

### 示例

调用 `fetch` 工具来获取一个网页内容：

```bash
ai-ops mcp call fetch.fetch '{"url":"https://www.example.com"}'
```

### 示例输出

```
调用MCP工具: fetch
================
参数: map[url:https://www.example.com]
正在执行...

✅ 调用成功!
结果:
----
{
  "content": "Example Domain...",
  "url": "https://www.example.com"
}

结果长度: 1256 字符
