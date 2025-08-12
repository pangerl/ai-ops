package llm

// Message 代表一个对话消息
type Message struct {
	Role       string     `json:"role"` // "user", "assistant", or "tool"
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"` // The name of the tool that was called
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // Only for role="tool"
}

// Response AI 响应结构
type Response struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        TokenUsage `json:"usage"`
	FinishReason string     `json:"finish_reason"`
}

// ToolCall 工具调用结构
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// TokenUsage 令牌使用统计
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ModelInfo 模型信息
type ModelInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	MaxTokens    int    `json:"max_tokens"`
	SupportTools bool   `json:"support_tools"`
}

// ClientManager has been deprecated and will be removed.
// All adapter lifecycle management is now handled by the llmRegistry.
