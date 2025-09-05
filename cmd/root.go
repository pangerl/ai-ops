package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ai-ops/internal/config"
	"ai-ops/internal/llm"
	"ai-ops/internal/tools"
	"ai-ops/internal/tools/plugins"
	"ai-ops/internal/util"
	"ai-ops/internal/util/errors"
)

var (
	// configPath 是配置文件的路径
	configPath string
	// verbose 标志用于启用详细输出
	verbose bool
	// llmManager 是全局的 AI 客户端管理器
	// llmManager is deprecated.
	toolManager tools.ToolManager
)

// rootCmd 代表没有调用子命令时的基础命令
var rootCmd = &cobra.Command{
	Use:   "llm-ops",
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
		return errors.WrapError(errors.ErrCodeConfigInvalid, "配置加载失败", err)
	}

	// 3. 根据verbose标志调整日志级别
	logLevel := config.Config.Logging.Level
	if verbose {
		logLevel = "debug"
	}

	// 4. 初始化日志系统
	logFormat := config.Config.Logging.Format
	if logFormat == "" {
		logFormat = "text"
	}
	logOutput := config.Config.Logging.Output
	if logOutput == "" {
		logOutput = "stdout"
	}
	logFile := config.Config.Logging.File

	if err := util.InitLogger(logLevel, logFormat, logOutput, logFile); err != nil {
		return errors.WrapError(errors.ErrCodeConfigInvalid, "日志系统初始化失败", err)
	}

	util.Info("应用配置加载完成")
	util.Debugw("配置详情", map[string]any{
		"default_model": config.Config.AI.DefaultModel,
		"log_level":     logLevel,
		"config_path":   configPath,
	})

	// 5. 初始化所有注册表并注册提供者
	if err := initializeRegistries(); err != nil {
		return errors.WrapError(errors.ErrCodeInitializationFailed, "注册表初始化失败", err)
	}

	// 6. 初始化 AI 客户端管理器
	if err := initializeAIClients(); err != nil {
		return errors.WrapError(errors.ErrCodeInitializationFailed, "AI 客户端初始化失败", err)
	}

	// 7. 初始化工具管理器和插件
	if err := initializeTools(); err != nil {
		return errors.WrapError(errors.ErrCodeInitializationFailed, "工具管理器初始化失败", err)
	}

	return nil
}

// initializeRegistries 初始化所有注册表
func initializeRegistries() error {
	// 初始化LLM注册表
	if err := llm.InitRegistry(); err != nil {
		return errors.WrapError(errors.ErrCodeInitializationFailed, "LLM注册表初始化失败", err)
	}
	// 显式注册所有 LLM 提供者
	if err := RegisterLLMProviders(); err != nil {
		return errors.WrapError(errors.ErrCodeInitializationFailed, "LLM 提供者注册失败", err)
	}

	// 初始化工具注册表
	if err := tools.InitRegistry(); err != nil {
		return errors.WrapError(errors.ErrCodeInitializationFailed, "工具注册表初始化失败", err)
	}
	return nil
}

// initializeTools 初始化工具管理器和插件
func initializeTools() error {
	var err error
	toolManager, err = tools.NewToolManager()
	if err != nil {
		return err
	}

	// 注册并初始化插件
	plugins.RegisterPluginFactories(toolManager)
	toolManager.InitializePlugins()

	util.Debugw("工具状态", map[string]interface{}{
		"registered_tools": len(toolManager.GetTools()),
	})
	return nil
}

// initializeAIClients 初始化 AI 客户端
func initializeAIClients() error {
	// 从配置中创建和注册客户端
	for name, modelConfig := range config.Config.AI.Models {
		if _, err := llm.CreateAdapter(name, modelConfig.Type, modelConfig); err != nil {
			util.Warnw(fmt.Sprintf("创建 AI 适配器 '%s' 失败", name), map[string]interface{}{"error": err})
			continue // 即使某个客户端失败，也继续尝试其他客户端
		}
		util.Debugw("AI 适配器已创建", map[string]interface{}{"name": name, "type": modelConfig.Type, "model": modelConfig.Model})
	}

	// 检查是否有任何客户端被成功创建
	if len(llm.ListAdapters()) == 0 {
		return errors.NewError(errors.ErrCodeInitializationFailed, "没有可用的 AI 适配器，请检查配置")
	}

	// 默认客户端的逻辑现在由使用方（例如 chat 命令）处理
	// 这里只打印信息
	defaultModelName := config.Config.AI.DefaultModel
	if _, exists := llm.GetAdapter(defaultModelName); !exists {
		util.Warnw(fmt.Sprintf("配置的默认模型 '%s' 不可用，将使用第一个可用模型", defaultModelName), nil)
		defaultModelName = llm.ListAdapters()[0]
	}

	util.Debugw("AI 适配器状态", map[string]interface{}{
		"registered_adapters": llm.ListAdapters(),
		"default_adapter":     defaultModelName,
	})

	return nil
}

// showStatus 显示应用状态
func showStatus() {
	util.Info("AI-Ops 应用启动成功")
	fmt.Println("AI-Ops 框架初始化完成")
	fmt.Println("配置文件加载成功")

	defaultModelName := config.Config.AI.DefaultModel
	if adapter, exists := llm.GetAdapter(defaultModelName); exists {
		fmt.Printf("默认AI模型: %s (%s)\n", adapter.GetModelInfo().Name, defaultModelName)
	} else if len(llm.ListAdapters()) > 0 {
		// 如果默认模型不存在，则显示第一个可用的模型
		firstAdapterName := llm.ListAdapters()[0]
		adapter, _ := llm.GetAdapter(firstAdapterName)
		fmt.Printf("默认AI模型: %s (%s) - (回退)\n", adapter.GetModelInfo().Name, firstAdapterName)
	} else {
		fmt.Println("默认AI模型: 未配置或初始化失败")
	}

	fmt.Printf("日志级别: %s\n", config.Config.Logging.Level)
	fmt.Println("\n使用 'llm-ops --help' 查看可用命令")
}
