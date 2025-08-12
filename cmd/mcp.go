package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"ai-ops/internal/mcp"
	"ai-ops/internal/tools"
	"ai-ops/internal/util/errors"
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP服务管理",
	Long:  "管理Model Context Protocol (MCP) 服务器和工具",
}

// mcpStatusCmd shows MCP service status
var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "显示MCP服务状态",
	Run: func(cmd *cobra.Command, args []string) {
		showMCPStatus()
	},
}

// mcpListCmd lists available MCP tools
var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用的MCP工具",
	Run: func(cmd *cobra.Command, args []string) {
		listMCPTools()
	},
}

// mcpTestCmd tests MCP connection
var mcpTestCmd = &cobra.Command{
	Use:   "test",
	Short: "测试MCP服务器连接",
	Run: func(cmd *cobra.Command, args []string) {
		testMCPConnection()
	},
}

// mcpCallCmd calls an MCP tool
var mcpCallCmd = &cobra.Command{
	Use:   "call [tool_name] [arguments_json]",
	Short: "调用MCP工具",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		callMCPTool(args)
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpStatusCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpTestCmd)
	mcpCmd.AddCommand(mcpCallCmd)
}

// withMCPService 是一个辅助函数，用于封装MCP服务的初始化和关闭逻辑
func withMCPService(run func(ctx context.Context, mcpService *mcp.MCPService, toolManager tools.ToolManager) error) error {
	toolManager, err := tools.NewToolManager()
	if err != nil {
		return err
	}
	mcpService := mcp.NewMCPService(toolManager, "mcp_settings.json", 30*time.Second)
	defer mcpService.Shutdown()

	ctx := context.Background()
	if err := mcpService.Initialize(ctx); err != nil {
		return err
	}

	return run(ctx, mcpService, toolManager)
}

// showMCPStatus 显示MCP服务状态
func showMCPStatus() {
	fmt.Println("MCP服务状态:")
	fmt.Println("============")

	err := withMCPService(func(ctx context.Context, mcpService *mcp.MCPService, toolManager tools.ToolManager) error {
		serverStatus := mcpService.GetServerStatus()
		connectedServers := mcpService.GetConnectedServers()

		fmt.Printf("配置文件: mcp_settings.json\n")
		fmt.Printf("已配置服务器数量: %d\n", len(serverStatus))
		fmt.Printf("已连接服务器数量: %d\n", len(connectedServers))

		if len(serverStatus) == 0 {
			fmt.Println("⚠️  未配置任何MCP服务器")
			return nil
		}

		fmt.Println("\n服务器状态:")
		for serverName, connected := range serverStatus {
			status := "❌ 未连接"
			if connected {
				status = "✅ 已连接"
			}
			fmt.Printf("  %s: %s\n", serverName, status)
		}

		toolDefs := toolManager.GetToolDefinitions()
		mcpToolCount := 0
		for _, toolDef := range toolDefs {
			if len(toolDef.Name) > 0 && toolDef.Description != "" &&
				(toolDef.Description[:5] == "[MCP:" || len(toolDef.Name) > 10) {
				mcpToolCount++
			}
		}

		fmt.Printf("\n已注册的MCP工具数量: %d\n", mcpToolCount)
		return nil
	})

	if err != nil {
		fmt.Printf("❌ 操作失败: %v\n", err)
	}
}

// listMCPTools 列出可用的MCP工具
func listMCPTools() {
	fmt.Println("可用的MCP工具:")
	fmt.Println("==============")

	err := withMCPService(func(ctx context.Context, mcpService *mcp.MCPService, toolManager tools.ToolManager) error {
		toolDefs := toolManager.GetToolDefinitions()

		if len(toolDefs) == 0 {
			fmt.Println("⚠️  未找到任何工具")
			return nil
		}

		serverTools := make(map[string][]tools.ToolDefinition)
		otherTools := []tools.ToolDefinition{}
		mcpToolRegex := regexp.MustCompile(`^\[MCP:([^\]]+)\]\s*`)

		for _, toolDef := range toolDefs {
			matches := mcpToolRegex.FindStringSubmatch(toolDef.Description)
			if len(matches) > 1 {
				serverName := matches[1]
				// 创建一个新的ToolDefinition，但移除描述中的前缀
				cleanToolDef := toolDef
				cleanToolDef.Description = strings.TrimSpace(mcpToolRegex.ReplaceAllString(toolDef.Description, ""))
				serverTools[serverName] = append(serverTools[serverName], cleanToolDef)
			} else {
				otherTools = append(otherTools, toolDef)
			}
		}

		mcpToolCount := 0
		for serverName, tools := range serverTools {
			fmt.Printf("\n服务器: %s\n", serverName)
			fmt.Println("--------")
			for _, tool := range tools {
				mcpToolCount++
				fmt.Printf("  • %s\n", tool.Name)
				if tool.Description != "" {
					fmt.Printf("    %s\n", tool.Description)
				}
			}
		}

		if len(otherTools) > 0 {
			fmt.Printf("\n其他工具:\n")
			fmt.Println("--------")
			for _, tool := range otherTools {
				fmt.Printf("  • %s - %s\n", tool.Name, tool.Description)
			}
		}

		fmt.Printf("\n总计: %d 个工具 (其中 %d 个MCP工具)\n",
			len(toolDefs), mcpToolCount)
		return nil
	})

	if err != nil {
		fmt.Printf("❌ 操作失败: %v\n", err)
	}
}

// testMCPConnection 测试MCP连接
func testMCPConnection() {
	fmt.Println("测试MCP服务器连接:")
	fmt.Println("==================")
	fmt.Println("正在初始化MCP服务...")

	err := withMCPService(func(ctx context.Context, mcpService *mcp.MCPService, toolManager tools.ToolManager) error {
		connectedServers := mcpService.GetConnectedServers()
		serverStatus := mcpService.GetServerStatus()

		fmt.Println("✅ MCP服务初始化成功")

		fmt.Printf("\n连接结果:\n")
		for serverName, connected := range serverStatus {
			if connected {
				fmt.Printf("  ✅ %s: 连接成功\n", serverName)
			} else {
				fmt.Printf("  ❌ %s: 连接失败\n", serverName)
			}
		}

		if len(connectedServers) > 0 {
			fmt.Printf("\n🎉 成功连接 %d 个服务器: %v\n",
				len(connectedServers), connectedServers)
		} else {
			fmt.Println("\n⚠️  没有成功连接任何服务器")
		}
		return nil
	})

	if err != nil {
		fmt.Printf("❌ 连接测试失败: %v\n", err)
		if errors.IsErrorCode(err, errors.ErrCodeConfigLoadFailed) {
			fmt.Println("\n💡 建议:")
			fmt.Println("  1. 检查 mcp_settings.json 文件是否存在")
			fmt.Println("  2. 验证JSON格式是否正确")
		} else if errors.IsErrorCode(err, errors.ErrCodeMCPConnectionFailed) {
			fmt.Println("\n💡 建议:")
			fmt.Println("  1. 检查MCP服务器命令是否正确")
			fmt.Println("  2. 确保相关依赖已安装 (如: uvx, uv)")
			fmt.Println("  3. 验证服务器程序是否可执行")
		}
	}
}

// callMCPTool 调用MCP工具
func callMCPTool(args []string) {
	if len(args) < 1 {
		fmt.Println("❌ 请指定工具名称")
		fmt.Println("用法: ai-ops mcp call [tool_name] [arguments_json]")
		return
	}

	toolName := args[0]
	var arguments map[string]any

	if len(args) > 1 {
		if err := json.Unmarshal([]byte(args[1]), &arguments); err != nil {
			fmt.Printf("❌ 参数解析失败: %v\n", err)
			fmt.Println("参数必须是有效的JSON格式，例如: '{\"url\":\"https://example.com\"}'")
			return
		}
	} else {
		arguments = make(map[string]any)
	}

	fmt.Printf("调用MCP工具: %s\n", toolName)
	fmt.Println("================")

	err := withMCPService(func(ctx context.Context, mcpService *mcp.MCPService, toolManager tools.ToolManager) error {
		toolCall := tools.ToolCall{
			ID:        fmt.Sprintf("mcp-call-%d", time.Now().Unix()),
			Name:      toolName,
			Arguments: arguments,
		}

		fmt.Printf("参数: %v\n", arguments)
		fmt.Println("正在执行...")

		result, err := toolManager.ExecuteToolCall(ctx, toolCall)
		if err != nil {
			return err
		}

		fmt.Println("\n✅ 调用成功!")
		fmt.Println("结果:")
		fmt.Println("----")

		var jsonResult interface{}
		if err := json.Unmarshal([]byte(result), &jsonResult); err == nil {
			if formatted, err := json.MarshalIndent(jsonResult, "", "  "); err == nil {
				fmt.Println(string(formatted))
			} else {
				fmt.Println(result)
			}
		} else {
			fmt.Println(result)
		}

		fmt.Printf("\n结果长度: %d 字符\n", len(result))
		return nil
	})

	if err != nil {
		fmt.Printf("❌ 工具调用失败: %v\n", err)
		if errors.IsErrorCode(err, errors.ErrCodeToolNotFound) {
			fmt.Println("\n💡 建议:")
			fmt.Println("  1. 使用 'ai-ops mcp list' 查看可用工具")
			fmt.Println("  2. 检查工具名称拼写是否正确")
		}
	}
}
