package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-ops/internal/util"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewMCPTool 创建新的MCP工具包装器
func NewMCPTool(serverName string, session *mcp.ClientSession, toolInfo *mcp.Tool, timeout time.Duration) *MCPTool {
	return &MCPTool{
		serverName: serverName,
		session:    session,
		toolInfo:   toolInfo,
		timeout:    timeout,
	}
}

// Name 获取工具名称
func (t *MCPTool) Name() string {
	return fmt.Sprintf("%s.%s", t.serverName, t.toolInfo.Name)
}

// Description 获取工具描述
func (t *MCPTool) Description() string {
	return fmt.Sprintf("[MCP:%s] %s", t.serverName, t.toolInfo.Description)
}

// Parameters 获取工具参数schema
func (t *MCPTool) Parameters() map[string]any {
	if t.toolInfo.InputSchema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	// 将InputSchema (any) 转换为 map[string]any
	var schema map[string]any
	data, err := json.Marshal(t.toolInfo.InputSchema)
	if err == nil {
		_ = json.Unmarshal(data, &schema)
	}

	// 递归修复schema
	if schema != nil {
		fixSchema(schema)
	}

	return schema
}

// fixSchema 递归修复JSON schema，以兼容Gemini API
func fixSchema(node map[string]any) {
	// 将 exclusiveMaximum/exclusiveMinimum 替换为 maximum/minimum
	if val, ok := node["exclusiveMaximum"]; ok {
		node["maximum"] = val
		delete(node, "exclusiveMaximum")
	}
	if val, ok := node["exclusiveMinimum"]; ok {
		node["minimum"] = val
		delete(node, "exclusiveMinimum")
	}

	// Gemini只支持string类型的 "enum" 和 "date-time" format
	if typeVal, ok := node["type"]; ok && typeVal == "string" {
		if formatVal, ok := node["format"]; ok {
			if s, ok := formatVal.(string); ok && s != "enum" && s != "date-time" {
				delete(node, "format")
			}
		}
	}

	// 递归处理子节点
	for _, value := range node {
		if subNode, ok := value.(map[string]any); ok {
			fixSchema(subNode)
		} else if subArray, ok := value.([]any); ok {
			for _, item := range subArray {
				if subNode, ok := item.(map[string]any); ok {
					fixSchema(subNode)
				}
			}
		}
	}
}

// Execute 执行工具
func (t *MCPTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	// 设置超时
	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	util.Infow("执行MCP工具", map[string]any{
		"server_name": t.serverName,
		"tool_name":   t.toolInfo.Name,
		"full_name":   t.Name(),
		"arguments":   args,
	})

	params := &mcp.CallToolParams{
		Name:      t.toolInfo.Name,
		Arguments: args,
	}

	result, err := t.session.CallTool(ctx, params)
	if err != nil {
		return "", util.WrapError(util.ErrCodeMCPToolCallFailed,
			fmt.Sprintf("MCP工具执行失败: %s", t.Name()), err)
	}

	if result.IsError {
		errMsg := "工具执行返回错误"
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
				errMsg = textContent.Text
			}
		}
		return "", util.NewError(util.ErrCodeMCPToolCallFailed,
			fmt.Sprintf("调用MCP工具失败: %s - %s", t.Name(), errMsg))
	}

	var resultStr string
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			resultStr += textContent.Text
		}
	}

	util.Infow("MCP工具执行成功", map[string]any{
		"server_name":   t.serverName,
		"tool_name":     t.toolInfo.Name,
		"full_name":     t.Name(),
		"result_length": len(resultStr),
	})

	return resultStr, nil
}
