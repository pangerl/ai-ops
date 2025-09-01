package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"ai-ops/internal/config"
	"ai-ops/internal/llm"
	"ai-ops/internal/react"
	"ai-ops/internal/util"
)

// reactCmd represents the react command
var reactCmd = &cobra.Command{
	Use:   "react [任务描述]",
	Short: "使用 ReAct 模式执行复杂任务",
	Long: `ReAct (Reasoning and Acting) 模式让AI能够：
1. 分析和推理问题
2. 调用工具获取信息
3. 基于结果继续推理
4. 最终完成复杂任务`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		task := args[0]

		modelName, _ := cmd.Flags().GetString("model")
		debug, _ := cmd.Flags().GetBool("debug")

		// 获取AI模型适配器
		var client llm.ModelAdapter
		var exists bool

		if modelName != "" {
			client, exists = llm.GetAdapter(modelName)
			if !exists {
				util.Error(fmt.Sprintf("指定的模型 '%s' 不存在或未正确配置。", modelName))
				return
			}
		} else {
			// 使用默认模型
			defaultModelName := config.Config.AI.DefaultModel
			client, exists = llm.GetAdapter(defaultModelName)
			if !exists {
				adapters := llm.ListAdapters()
				if len(adapters) == 0 {
					util.Error("没有可用的AI适配器。请检查您的配置。")
					return
				}
				defaultModelName = adapters[0]
				client, _ = llm.GetAdapter(defaultModelName)
			}
		}

		// 创建ReAct智能体
		agent := react.NewAgent(client, toolManager)

		// 执行任务
		ctx := context.Background()
		result, err := agent.Execute(ctx, task, debug)
		if err != nil {
			util.Error(fmt.Sprintf("执行ReAct任务失败: %v", err))
			return
		}

		fmt.Println("\n=== 最终结果 ===")
		fmt.Println(result)
	},
}

func init() {
	rootCmd.AddCommand(reactCmd)

	reactCmd.Flags().StringP("model", "m", "", "指定使用的AI模型")
	reactCmd.Flags().BoolP("debug", "d", false, "显示详细的推理过程")
}
