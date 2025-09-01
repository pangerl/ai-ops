package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	util "ai-ops/internal/util"

	"github.com/BurntSushi/toml"
)

// 全局配置实例
var Config *AppConfig

// 应用配置结构
type AppConfig struct {
	AI      AIConfig      `toml:"ai"`
	Logging LoggingConfig `toml:"logging"`
	Weather WeatherConfig `toml:"weather"`
	RAG     RAGConfig     `toml:"rag"`
	Tools   ToolsConfig   `toml:"tools"`
}

// AI配置
type AIConfig struct {
	DefaultModel string                 `toml:"default_model"`
	Models       map[string]ModelConfig `toml:"models"`
	Timeout      int                    `toml:"timeout"` // 超时时间（秒）
}

// 模型配置
type ModelConfig struct {
	Type    string `toml:"type"` // "gemini" 或 "openai"
	APIKey  string `toml:"api_key"`
	BaseURL string `toml:"base_url"`
	Model   string `toml:"model"`
	Style   string `toml:"style" json:"style,omitempty"`
}

// 日志配置
type LoggingConfig struct {
	Level  string `toml:"level"`  // debug, info, warn, error
	Format string `toml:"format"` // json, text
	Output string `toml:"output"` // stdout, stderr, file
	File   string `toml:"file"`   // 日志文件路径
}

// 天气配置
type WeatherConfig struct {
	ApiHost string `toml:"api_host"`
	ApiKey  string `toml:"api_key"`
}

// RAG配置
type RAGConfig struct {
	Enable     bool   `toml:"enable"`
	ApiHost    string `toml:"api_host"`
	RetrievalK int    `toml:"retrieval_k"`
	TopK       int    `toml:"top_k"`
}

// 工具配置
type ToolsConfig struct {
	Echo    bool `toml:"echo"`    // Echo工具（调试用）
	Weather bool `toml:"weather"` // 天气工具
	RAG     bool `toml:"rag"`     // RAG工具
}

// 加载配置文件
func LoadConfig(configPath string) error {
	// 如果没有指定配置文件路径，使用默认路径
	if configPath == "" {
		configPath = getDefaultConfigPath()
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建默认配置文件
		if err := createDefaultConfig(configPath); err != nil {
			return fmt.Errorf("创建默认配置文件失败: %w", err)
		}
		util.Infow("已创建默认配置文件", map[string]interface{}{"path": configPath})
	}

	// 解析TOML配置文件
	var config AppConfig
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 使用环境变量覆盖配置
	overrideWithEnv(&config)

	// 验证配置
	if err := validateConfig(&config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 设置全局配置
	Config = &config
	return nil
}

// 获取默认配置文件路径
func getDefaultConfigPath() string {
	// 优先使用当前目录下的 config.toml
	if _, err := os.Stat("config.toml"); err == nil {
		return "config.toml"
	}

	// 使用用户主目录下的配置文件
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}

	return filepath.Join(homeDir, ".ai-ops", "config.toml")
}

// 创建默认配置文件
func createDefaultConfig(configPath string) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 默认配置内容
	defaultConfig := `# AI-Ops 配置文件

[ai]
default_model = "gemini"
timeout = 30

[ai.models.gemini]
type = "gemini"
api_key = "${GEMINI_API_KEY}"
base_url = "https://generativelanguage.googleapis.com"
model = "gemini-pro"

[ai.models.openai]
type = "openai"
api_key = "${OPENAI_API_KEY}"
base_url = "https://api.openai.com"
model = "gpt-3.5-turbo"

[logging]
level = "info"
format = "text"
output = "stdout"
file = ""

[weather]
api_host = "https://devapi.qweather.com"
api_key = "${QWEATHER_API_KEY}"

[rag]
api_host = "http://localhost:8000"
retrieval_k = 15
top_k = 5

[tools]
# 工具启用配置（sysinfo 为核心工具，始终启用）
echo = false     # Echo工具（调试用）
weather = false  # 天气工具（需要配置 QWEATHER_API_KEY）
rag = false      # RAG工具（需要启动RAG服务）
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

// 使用环境变量覆盖配置
func overrideWithEnv(config *AppConfig) {
	// AI模型配置
	for name, model := range config.AI.Models {
		if apiKey := getEnvForModel(name, "API_KEY"); apiKey != "" {
			model.APIKey = apiKey
		}
		if baseURL := getEnvForModel(name, "BASE_URL"); baseURL != "" {
			model.BaseURL = baseURL
		}
		if style := getEnvForModel(name, "STYLE"); style != "" {
			model.Style = style
		}
		// 一次性写回，避免多次赋值
		config.AI.Models[name] = model
	}

	// 天气API配置
	if apiKey := os.Getenv("QWEATHER_API_KEY"); apiKey != "" {
		config.Weather.ApiKey = apiKey
	}
	if apiHost := os.Getenv("QWEATHER_API_HOST"); apiHost != "" {
		config.Weather.ApiHost = apiHost
	}

	// 日志配置
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}
	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		config.Logging.Output = output
	}
	if file := os.Getenv("LOG_FILE"); file != "" {
		config.Logging.File = file
	}
}

// 获取模型相关的环境变量
func getEnvForModel(modelName, suffix string) string {
	// 尝试多种环境变量命名格式
	envNames := []string{
		fmt.Sprintf("%s_%s", strings.ToUpper(modelName), suffix),
		fmt.Sprintf("AI_OPS_%s_%s", strings.ToUpper(modelName), suffix),
	}

	for _, envName := range envNames {
		if value := os.Getenv(envName); value != "" {
			return value
		}
	}

	return ""
}

// 验证配置
func validateConfig(config *AppConfig) error {
	// 验证AI配置
	if err := validateAIConfig(&config.AI); err != nil {
		return fmt.Errorf("AI配置验证失败: %w", err)
	}

	// 验证日志配置
	if err := validateLoggingConfig(&config.Logging); err != nil {
		return fmt.Errorf("日志配置验证失败: %w", err)
	}

	// 验证天气配置（可选，但如果配置了需要验证）
	if config.Weather.ApiHost != "" || config.Weather.ApiKey != "" {
		if err := validateWeatherConfig(&config.Weather); err != nil {
			return fmt.Errorf("天气配置验证失败: %w", err)
		}
	}

	// 验证RAG配置（如果启用）
	if config.RAG.Enable {
		if err := validateRAGConfig(&config.RAG); err != nil {
			return fmt.Errorf("RAG配置验证失败: %w", err)
		}
	}

	return nil
}

// 验证AI配置
func validateAIConfig(aiConfig *AIConfig) error {
	// 验证默认模型是否存在
	if aiConfig.DefaultModel == "" {
		return fmt.Errorf("默认AI模型未配置")
	}

	if _, exists := aiConfig.Models[aiConfig.DefaultModel]; !exists {
		return fmt.Errorf("默认AI模型 '%s' 未在models中定义", aiConfig.DefaultModel)
	}

	// 验证超时配置
	if aiConfig.Timeout < 0 || aiConfig.Timeout > 300 {
		return fmt.Errorf("AI超时配置不合理: %d秒（应在0-300秒之间）", aiConfig.Timeout)
	}

	// 验证每个模型配置
	for name, model := range aiConfig.Models {
		if err := validateModelConfig(name, &model); err != nil {
			return fmt.Errorf("模型 '%s' 配置验证失败: %w", name, err)
		}
	}

	return nil
}

// 验证单个模型配置
func validateModelConfig(name string, model *ModelConfig) error {
	// 验证模型类型
	validTypes := []string{"openai", "gemini"}
	typeValid := false
	for _, validType := range validTypes {
		if model.Type == validType {
			typeValid = true
			break
		}
	}
	if !typeValid {
		return fmt.Errorf("不支持的模型类型: %s", model.Type)
	}

	// 验证API密钥（允许环境变量占位符）
	if model.APIKey == "" || (model.APIKey != "" && !strings.HasPrefix(model.APIKey, "${") && len(model.APIKey) < 10) {
		util.Warnw("模型API密钥可能无效", map[string]interface{}{
			"model": name,
		})
	}

	// 验证BaseURL格式（如果提供）
	if model.BaseURL != "" && !strings.HasPrefix(model.BaseURL, "http") {
		return fmt.Errorf("BaseURL格式不正确，必须以http或https开头: %s", model.BaseURL)
	}

	// 验证模型名称
	if model.Model == "" {
		return fmt.Errorf("模型配置'%s'的模型名称不能为空", name)
	}

	return nil
}

// 验证日志配置
func validateLoggingConfig(logging *LoggingConfig) error {
	// 验证日志级别
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("无效的日志级别: %s", logging.Level)
	}

	// 验证日志格式
	validFormats := []string{"text", "json"}
	formatValid := false
	for _, format := range validFormats {
		if logging.Format == format {
			formatValid = true
			break
		}
	}
	if !formatValid {
		return fmt.Errorf("无效的日志格式: %s", logging.Format)
	}

	// 验证日志输出
	validOutputs := []string{"stdout", "stderr", "file"}
	outputValid := false
	for _, output := range validOutputs {
		if logging.Output == output {
			outputValid = true
			break
		}
	}
	if !outputValid {
		return fmt.Errorf("无效的日志输出: %s", logging.Output)
	}

	// 如果输出到文件，验证文件路径
	if logging.Output == "file" && logging.File == "" {
		return fmt.Errorf("日志输出设置为文件但未指定文件路径")
	}

	return nil
}

// 验证天气配置
func validateWeatherConfig(weather *WeatherConfig) error {
	if weather.ApiHost == "" {
		return fmt.Errorf("天气API主机未配置")
	}

	if !strings.HasPrefix(weather.ApiHost, "http") {
		return fmt.Errorf("天气API主机格式不正确: %s", weather.ApiHost)
	}

	if weather.ApiKey == "" || (weather.ApiKey != "" && !strings.HasPrefix(weather.ApiKey, "${") && len(weather.ApiKey) < 10) {
		util.Warnw("天气API密钥可能无效", nil)
	}

	return nil
}

// 验证RAG配置
func validateRAGConfig(rag *RAGConfig) error {
	if rag.ApiHost == "" {
		return fmt.Errorf("RAG API主机未配置")
	}

	if !strings.HasPrefix(rag.ApiHost, "http") {
		return fmt.Errorf("RAG API主机格式不正确: %s", rag.ApiHost)
	}

	if rag.RetrievalK <= 0 || rag.RetrievalK > 100 {
		return fmt.Errorf("RAG检索数量配置不合理: %d（应在1-100之间）", rag.RetrievalK)
	}

	if rag.TopK <= 0 || rag.TopK > rag.RetrievalK {
		return fmt.Errorf("RAG顶部结果数量配置不合理: %d（应在1-%d之间）", rag.TopK, rag.RetrievalK)
	}

	return nil
}

// 获取当前配置
func GetConfig() *AppConfig {
	return Config
}

// 获取指定模型的配置
func GetModelConfig(modelName string) (ModelConfig, error) {
	if Config == nil {
		return ModelConfig{}, fmt.Errorf("配置未初始化")
	}

	if modelName == "" {
		modelName = Config.AI.DefaultModel
	}

	model, exists := Config.AI.Models[modelName]
	if !exists {
		return ModelConfig{}, fmt.Errorf("模型 '%s' 未配置", modelName)
	}

	return model, nil
}
