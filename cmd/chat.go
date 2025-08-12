package cmd

import (
	"context"
	"fmt"
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
	Long:  "启动与AI助手的交互式对话，支持工具调用和上下文管理",
	Run: func(cmd *cobra.Command, args []string) {
		util.Info("正在启动交互式对话模式...")

		modelName, _ := cmd.Flags().GetString("model")
		var client llm.ModelAdapter
		var exists bool

		if modelName != "" {
			client, exists = llm.GetAdapter(modelName)
			if !exists {
				util.Error(fmt.Sprintf("指定的模型 '%s' 不存在或未正确配置。", modelName))
				return
			}
			util.Infow("已切换到指定模型", map[string]any{"model": modelName})
		} else {
			// 从配置中获取默认模型
			defaultModelName := config.Config.AI.DefaultModel
			client, exists = llm.GetAdapter(defaultModelName)
			if !exists {
				// 如果默认模型不存在，则使用第一个可用的模型
				adapters := llm.ListAdapters()
				if len(adapters) == 0 {
					util.Error("没有可用的AI适配器。请检查您的配置。")
					return
				}
				defaultModelName = adapters[0]
				client, _ = llm.GetAdapter(defaultModelName)
				util.Warnw(fmt.Sprintf("默认模型不可用，回退到 %s", defaultModelName), nil)
			}
			util.Infow("使用默认模型", map[string]any{"model": client.GetModelInfo().Name})
		}

		// 初始化MCP服务
		// 全局的 toolManager 实例已在 root.go 中初始化
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

		// 注意：RunSimpleLoop 现在可能需要调整以使用新的服务层
		// 暂时保持不变，但假设它现在可以处理 ModelAdapter
		chat.RunSimpleLoop(client, toolManager)

		util.Info("对话模式已退出。")
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)

	// 对话命令标志
	chatCmd.Flags().StringP("model", "m", "", "指定使用的AI模型 (例如: openai, gemini)")
}
