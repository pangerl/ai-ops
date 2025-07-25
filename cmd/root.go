package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ai-ops/internal/ai"
	"ai-ops/internal/config"
	"ai-ops/internal/util"
)

var (
	// configPath 是配置文件的路径
	configPath string
	// verbose 标志用于启用详细输出
	verbose bool
	// aiManager 是全局的 AI 客户端管理器
	aiManager *ai.ClientManager
)

// rootCmd 代表没有调用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "ai-ops",
	Short: "AI-Ops 智能运维工具",
	Long: `AI-Ops 是一个基于人工智能的运维工具，
提供智能对话、工具调用和自动化运维功能。`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeApp()
	},
	Run: func(cmd *cobra.Command, args []string) {
		// 默认行为：显示状态信息
		showStatus()
	},
}

// Execute 将所有子命令添加到根命令并适当设置标志。
// 这是由 main.main() 调用的。它只需要对 rootCmd 调用一次。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "命令执行失败: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "配置文件路径 (默认: $AI_OPS_CONFIG)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "详细输出")
}

// initializeApp 初始化应用
func initializeApp() error {
	// 1. 处理配置文件路径
	if configPath == "" {
		configPath = os.Getenv("AI_OPS_CONFIG")
	}

	// 2. 加载配置文件
	if err := config.LoadConfig(configPath); err != nil {
		return util.WrapError(util.ErrCodeConfigInvalid, "配置加载失败", err)
	}

	// 3. 根据verbose标志调整日志级别
	logLevel := config.Config.Logging.Level
	if verbose {
		logLevel = "debug"
	}

	// 4. 初始化日志系统
	logConfig := config.Config.Logging
	if err := util.InitLogger(logLevel, logConfig.Format, logConfig.Output, logConfig.File); err != nil {
		return util.WrapError(util.ErrCodeConfigInvalid, "日志系统初始化失败", err)
	}

	util.Info("应用配置加载完成")
	util.Debugw("配置详情", map[string]any{
		"default_model": config.Config.AI.DefaultModel,
		"log_level":     logLevel,
		"config_path":   configPath,
	})

	// 5. 初始化 AI 客户端管理器
	if err := initializeAIClients(); err != nil {
		return util.WrapError(util.ErrCodeInitializationFailed, "AI 客户端初始化失败", err)
	}

	return nil
}

// initializeAIClients 初始化 AI 客户端
func initializeAIClients() error {
	aiManager = ai.NewClientManager()

	// 从配置中创建和注册客户端
	for name, modelConfig := range config.Config.AI.Models {
		// 将 config.ModelConfig 转换为 ai.ModelConfig
		aiModelConfig := ai.ModelConfig{
			Type:    modelConfig.Type,
			APIKey:  modelConfig.APIKey,
			BaseURL: modelConfig.BaseURL,
			Model:   modelConfig.Model,
			Timeout: config.Config.AI.Timeout,
		}

		if err := aiManager.CreateClientFromConfig(name, aiModelConfig); err != nil {
			util.Warnw(fmt.Sprintf("创建 AI 客户端 '%s' 失败", name), map[string]interface{}{"error": err})
			continue // 即使某个客户端失败，也继续尝试其他客户端
		}
		util.Debugw("AI 客户端已创建", map[string]interface{}{"name": name, "type": aiModelConfig.Type, "model": aiModelConfig.Model})
	}

	// 检查是否有任何客户端被成功创建
	if len(aiManager.ListClients()) == 0 {
		return util.NewError(util.ErrCodeInitializationFailed, "没有可用的 AI 客户端，请检查配置")
	}

	// 设置默认客户端
	defaultModel := config.Config.AI.DefaultModel
	if defaultModel != "" {
		if err := aiManager.SetDefaultClient(defaultModel); err != nil {
			util.Warnw(fmt.Sprintf("设置默认模型 '%s' 失败，将使用第一个可用模型", defaultModel), map[string]interface{}{"error": err})
		}
	}

	util.Info("AI 客户端初始化完成")
	util.Debugw("AI 客户端状态", map[string]interface{}{
		"registered_clients": aiManager.ListClients(),
		"default_client":     aiManager.GetDefaultClient().GetModelInfo().Name,
	})

	return nil
}

// showStatus 显示应用状态
func showStatus() {
	util.Info("AI-Ops 应用启动成功")
	fmt.Println("AI-Ops 框架初始化完成")
	fmt.Println("配置文件加载成功")

	if aiManager != nil && aiManager.GetDefaultClient() != nil {
		fmt.Printf("默认AI模型: %s (%s)\n", aiManager.GetDefaultClient().GetModelInfo().Name, config.Config.AI.DefaultModel)
	} else {
		fmt.Println("默认AI模型: 未配置或初始化失败")
	}

	fmt.Printf("日志级别: %s\n", config.Config.Logging.Level)
	fmt.Println("\n使用 'ai-ops --help' 查看可用命令")
}
