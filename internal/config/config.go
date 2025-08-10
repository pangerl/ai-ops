package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		fmt.Printf("已创建默认配置文件: %s\n", configPath)
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
	// 优先使用当前目录下的configs/config.toml
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
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

// 使用环境变量覆盖配置
func overrideWithEnv(config *AppConfig) {
	// AI模型配置
	for name, model := range config.AI.Models {
		if apiKey := getEnvForModel(name, "API_KEY"); apiKey != "" {
			model.APIKey = apiKey
			config.AI.Models[name] = model
		}
		if baseURL := getEnvForModel(name, "BASE_URL"); baseURL != "" {
			model.BaseURL = baseURL
			config.AI.Models[name] = model
		}
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
	// 验证默认模型是否存在
	if config.AI.DefaultModel == "" {
		return fmt.Errorf("默认AI模型未配置")
	}

	if _, exists := config.AI.Models[config.AI.DefaultModel]; !exists {
		return fmt.Errorf("默认AI模型 '%s' 未在models中定义", config.AI.DefaultModel)
	}

	// 验证日志级别
	validLevels := []string{"debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if config.Logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return fmt.Errorf("无效的日志级别: %s", config.Logging.Level)
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
