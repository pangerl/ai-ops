package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"ai-ops/internal/chat"
	"ai-ops/internal/config"
	"ai-ops/internal/llm"
	"ai-ops/internal/mcp"
	"ai-ops/internal/util"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "启动交互式对话模式",
	Long: `启动与AI助手的交互式对话。

模式说明:
  普通模式: 友好的AI助手，专注问答和代码协助
  智能体模式: 自主规划任务，适合复杂运维场景

使用示例:
  ai-ops chat              # 普通对话模式
  ai-ops chat -a           # 智能体模式
  ai-ops chat -a -t        # 智能体模式 + 显示思考过程`,
	Run: func(cmd *cobra.Command, args []string) {
		util.Info("正在启动交互式对话模式...")

		// 获取默认模型
		client := getDefaultClient()
		if client == nil {
			util.Error("没有可用的AI模型配置，请检查config.toml")
			return
		}

		// 解析参数
		isAgent, _ := cmd.Flags().GetBool("agent")
		showThinking, _ := cmd.Flags().GetBool("think")

		// 创建会话配置
		sessionConfig := chat.SessionConfig{
			Mode:         getMode(isAgent),
			ShowThinking: showThinking,
		}

		// 初始化MCP服务
		mcpService := mcp.NewMCPService(toolManager, "mcp_settings.json", 30*time.Second)
		ctx := context.Background()

		if err := mcpService.Initialize(ctx); err != nil {
			util.Warnw("MCP服务初始化失败，将继续使用其他工具", map[string]any{
				"error": err.Error(),
			})
		} else {
			// 确保在程序退出时清理MCP资源
			defer func() {
				if err := mcpService.Shutdown(); err != nil {
					util.Warnw("MCP服务关闭失败", map[string]any{
						"error": err.Error(),
					})
				}
			}()

			connectedServers := mcpService.GetConnectedServers()
			if len(connectedServers) > 0 {
				util.Debugw("MCP服务初始化成功", map[string]any{
					"connected_servers": connectedServers,
				})
			}
		}

		// 启动对话
		chat.RunChat(client, toolManager, sessionConfig)

		util.Info("对话模式已退出。")
	},
}

// getDefaultClient 获取默认配置的AI客户端
func getDefaultClient() llm.ModelAdapter {
	defaultModel := config.Config.AI.DefaultModel
	client, exists := llm.GetAdapter(defaultModel)
	if !exists {
		// 尝试获取任何可用的适配器
		adapters := llm.ListAdapters()
		if len(adapters) == 0 {
			return nil
		}
		client, _ = llm.GetAdapter(adapters[0])
	}
	return client
}

// getMode 根据agent参数确定模式
func getMode(isAgent bool) string {
	if isAgent {
		return "agent"
	}
	return "chat"
}

func init() {
	rootCmd.AddCommand(chatCmd)

	// 对话命令参数
	chatCmd.Flags().BoolP("agent", "a", false, "启用智能体模式")
	chatCmd.Flags().BoolP("think", "t", false, "显示AI思考过程")
}
