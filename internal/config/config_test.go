package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	configContent := `
[ai]
default_model = "test_model"
timeout = 60

[ai.models.test_model]
type = "openai"
api_key = "test_key"
base_url = "https://test.api.com"
model = "test-model"

[cli]
history_size = 50
prompt = "test> "
exit_commands = ["exit"]

[logging]
level = "debug"
format = "json"
output = "stdout"

[weather]
api_host = "https://test.weather.com"
api_key = "test_weather_key"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	// 测试加载配置
	err = LoadConfig(configPath)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置值
	if Config.AI.DefaultModel != "test_model" {
		t.Errorf("期望默认模型为 'test_model'，实际为 '%s'", Config.AI.DefaultModel)
	}

	if Config.AI.Timeout != 60 {
		t.Errorf("期望超时时间为 60，实际为 %d", Config.AI.Timeout)
	}

	if Config.CLI.HistorySize != 50 {
		t.Errorf("期望历史记录大小为 50，实际为 %d", Config.CLI.HistorySize)
	}

	if Config.Logging.Level != "debug" {
		t.Errorf("期望日志级别为 'debug'，实际为 '%s'", Config.Logging.Level)
	}
}

func TestGetModelConfig(t *testing.T) {
	// 设置测试配置
	Config = &AppConfig{
		AI: AIConfig{
			DefaultModel: "test_model",
			Models: map[string]ModelConfig{
				"test_model": {
					Type:    "openai",
					APIKey:  "test_key",
					BaseURL: "https://test.api.com",
					Model:   "test-model",
				},
			},
		},
	}

	// 测试获取默认模型配置
	model, err := GetModelConfig("")
	if err != nil {
		t.Fatalf("获取默认模型配置失败: %v", err)
	}

	if model.Type != "openai" {
		t.Errorf("期望模型类型为 'openai'，实际为 '%s'", model.Type)
	}

	// 测试获取指定模型配置
	model, err = GetModelConfig("test_model")
	if err != nil {
		t.Fatalf("获取指定模型配置失败: %v", err)
	}

	if model.APIKey != "test_key" {
		t.Errorf("期望API密钥为 'test_key'，实际为 '%s'", model.APIKey)
	}

	// 测试获取不存在的模型配置
	_, err = GetModelConfig("nonexistent")
	if err == nil {
		t.Error("期望获取不存在的模型配置时返回错误")
	}
}
