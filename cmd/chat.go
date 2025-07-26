package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"ai-ops/internal/ai"
	"ai-ops/internal/chat"
	"ai-ops/internal/tools"
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

		// 初始化工具管理器
		toolManager := tools.NewToolManager()

		// 从插件注册表中创建所有自动注册的工具
		pluginTools := tools.CreatePluginTools()

		// 将插件工具注册到管理器
		for _, tool := range pluginTools {
			if err := toolManager.RegisterTool(tool); err != nil {
				// 仅记录警告，而不是中止程序
				util.Warnw("注册工具失败，已跳过", map[string]any{
					"tool_name": tool.Name(),
					"error":     err.Error(),
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
