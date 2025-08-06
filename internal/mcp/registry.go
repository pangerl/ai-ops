package mcp

import (
	"context"
	"time"

	"ai-ops/internal/tools"
	"ai-ops/internal/util"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPToolRegistry MCP工具注册器
type MCPToolRegistry struct {
	manager     MCPManager
	toolManager tools.ToolManager
	timeout     time.Duration
}

// NewMCPToolRegistry 创建新的MCP工具注册器
func NewMCPToolRegistry(manager MCPManager, toolManager tools.ToolManager, timeout time.Duration) *MCPToolRegistry {
	return &MCPToolRegistry{
		manager:     manager,
		toolManager: toolManager,
		timeout:     timeout,
	}
}

// RegisterMCPTools 注册所有MCP工具
func (r *MCPToolRegistry) RegisterMCPTools(ctx context.Context) error {
	util.Infow("开始注册MCP工具", nil)

	sessions := r.manager.GetClients()
	totalTools := 0

	for serverName, session := range sessions {
		// 获取工具列表
		result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			util.LogErrorWithFields(err, "获取MCP工具列表失败", map[string]any{
				"server_name": serverName,
			})
			continue
		}

		// 注册每个工具
		for _, toolInfo := range result.Tools {
			mcpTool := NewMCPTool(serverName, session, toolInfo, r.timeout)

			if err := r.toolManager.RegisterTool(mcpTool); err != nil {
				util.LogErrorWithFields(err, "注册MCP工具失败", map[string]any{
					"server_name": serverName,
					"tool_name":   toolInfo.Name,
					"full_name":   mcpTool.Name(),
				})
				continue
			}

			totalTools++
			util.Infow("MCP工具注册成功", map[string]any{
				"server_name": serverName,
				"tool_name":   toolInfo.Name,
				"full_name":   mcpTool.Name(),
			})
		}
	}

	util.Infow("MCP工具注册完成", map[string]any{
		"total_tools":  totalTools,
		"server_count": len(sessions),
	})

	return nil
}

// RefreshMCPTools 刷新MCP工具注册
func (r *MCPToolRegistry) RefreshMCPTools(ctx context.Context) error {
	util.Infow("刷新MCP工具注册", nil)

	// 重新初始化客户端
	if err := r.manager.InitializeClients(ctx); err != nil {
		return util.WrapError(util.ErrCodeMCPConnectionFailed, "重新初始化MCP客户端失败", err)
	}

	// 重新注册工具
	return r.RegisterMCPTools(ctx)
}
