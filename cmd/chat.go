package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"ai-ops/internal/ai"
	"ai-ops/internal/chat"
	"ai-ops/internal/mcp"
	"ai-ops/internal/tools"
	_ "ai-ops/internal/tools/plugins" // 匿名导入以触发插件注册
	"ai-ops/internal/util"
)

// chatCmd represents the chat command
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "启动交互式对话模式",
	Long:  "启动与AI助手的交互式对话，支持工具调用和上下文管理",
	Run: func(cmd *cobra.Command, args []string) {
		util.Info("正在启动交互式对话模式...")

		modelName, _ := cmd.Flags().GetString("model")
		var client ai.AIClient

		if modelName != "" {
			var exists bool
			client, exists = aiManager.GetClient(modelName)
			if !exists {
				util.Error(fmt.Sprintf("指定的模型 '%s' 不存在或未正确配置。", modelName))
				return
			}
			util.Infow("已切换到指定模型", map[string]any{"model": modelName})
		} else {
			client = aiManager.GetDefaultClient()
			if client == nil {
				util.Error("没有可用的AI客户端。请检查您的配置。")
				return
			}
			util.Infow("使用默认模型", map[string]any{"model": client.GetModelInfo().Name})
		}

		// 使用全局的默认工具管理器
		toolManager := tools.DefaultManager
		// 初始化所有通过工厂注册的插件
		toolManager.InitializePlugins()

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
				util.Infow("MCP服务初始化成功", map[string]any{
					"connected_servers": connectedServers,
				})
			}
		}

		chat.RunSimpleLoop(client, toolManager)

		util.Info("对话模式已退出。")
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)

	// 对话命令标志
	chatCmd.Flags().StringP("model", "m", "", "指定使用的AI模型 (例如: openai, gemini)")
}
