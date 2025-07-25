package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"ai-ops/internal/config"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long:  "管理AI-Ops的配置文件和设置",
}

// configShowCmd represents the show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	Run: func(cmd *cobra.Command, args []string) {
		showConfig()
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
}

// showConfig 显示配置信息
func showConfig() {
	fmt.Println("当前配置:")
	fmt.Printf("  配置文件: %s\n", configPath)
	fmt.Printf("  AI模型: %s\n", config.Config.AI.DefaultModel)
	fmt.Printf("  日志级别: %s\n", config.Config.Logging.Level)

	if verbose {
		// 检查默认模型的API密钥是否已配置
		if modelConfig, exists := config.Config.AI.Models[config.Config.AI.DefaultModel]; exists {
			fmt.Printf("  API密钥已配置: %t\n", modelConfig.APIKey != "")
		}
		fmt.Printf("  日志格式: %s\n", config.Config.Logging.Format)
		fmt.Printf("  日志输出: %s\n", config.Config.Logging.Output)
	}
}
