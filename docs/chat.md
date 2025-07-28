# 内部聊天模块 (`internal/chat`)

该模块负责处理用户与 AI 之间的交互逻辑。

## 结构

- `tui.go`: 提供了基于终端的用户界面 (Terminal User Interface)。它负责接收用户输入、显示 AI 回复以及处理 UI 相关的元素（如加载动画和颜色）。它不包含任何核心的对话逻辑。
- `session.go`: 定义了 `Session` 结构体，用于管理一个完整的对话会话。这是对话逻辑的核心。

## 工作流程

1.  **启动**: `RunSimpleLoop` (在 `tui.go` 中) 被调用，它初始化 TUI 并创建一个 `Session` 实例。
2.  **用户输入**: TUI 捕获用户在命令行中的输入。
3.  **处理消息**: TUI 将用户输入传递给 `session.ProcessMessage` 方法。
4.  **AI 调用循环 (在 `session.go` 中)**:
    a.  用户的消息被添加到会话的历史记录中。
    b.  **历史记录管理**: 在调用 AI 之前，`trimHistory` 方法会检查历史记录的长度。如果超过预设的 `maxHistory`，它会保留第一条消息和最近的 `maxHistory - 1` 条消息，以防止上下文无限增长。
    c.  `Session` 将修剪后的消息历史发送给 AI (通过 `ai.AIClient` 接口)。
    d.  `Session` 检查 AI 响应中的 `FinishReason` 字段。
    e.  **如果 `FinishReason` 是 `stop`**:
        -   表示对话此轮已完成。
        -   `ProcessMessage` 返回最终的文本响应。
    f.  **如果 `FinishReason` 是 `tool_calls`**:
        -   表示 AI 请求调用一个或多个工具。
        -   `Session` 调用 `toolManager.ExecuteToolCall` 来执行每个工具。
        -   工具的返回结果被格式化成 `role="tool"` 的消息，并添加到会话历史记录中。
        -   循环返回步骤 c，再次调用 AI，并附带上工具调用的结果。
5.  **显示结果**: `ProcessMessage` 方法返回最终的 AI 响应后，`tui.go` 中的 `RunSimpleLoop` 将其打印到控制台。

## 设计思想

- **关注点分离 (Separation of Concerns)**: `tui.go` 只关心“如何显示”，而 `session.go` 只关心“如何处理对话”。这种分离使得代码更易于维护和测试。我们可以独立地修改 UI 而不影响核心逻辑，反之亦然。
- **状态管理**: `Session` 结构体封装了所有与对话相关的状态，包括消息历史。这使得 `tui.go` 可以保持无状态，专注于 UI 渲染。
- **高效的上下文管理**: 通过 `trimHistory` 方法，`Session` 实现了“固定窗口 + 上下文保留”的策略。这既能通过保留初始上下文来维持对话的连贯性，又能有效控制 API 调用成本和避免超出模型的上下文长度限制。
- **可扩展的工具使用**: 工具调用的逻辑被完全封装在 `Session` 中。它遵循了 OpenAI 的函数调用模型，可以透明地处理需要多步工具调用的复杂场景。
