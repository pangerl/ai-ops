package cmd

import (
	"ai-ops/internal/llm"
	"ai-ops/internal/util"
	"fmt"
)

// RegisterLLMProviders 显式注册所有支持的 LLM 适配器工厂。
// 这个函数取代了之前使用 init() 函数的隐式注册机制。
func RegisterLLMProviders() error {
	util.Debug("开始注册 LLM 提供者...")

	// 注册 OpenAI 适配器
	if err := llm.RegisterAdapterFactory("openai", llm.NewOpenAIAdapter, llm.OpenAIAdapterInfo); err != nil {
		return fmt.Errorf("无法注册 openai 适配器工厂: %v", err)
	}
	util.Debug("OpenAI 提供者已注册")

	// 注册 Gemini 适配器
	if err := llm.RegisterAdapterFactory("gemini", llm.NewGeminiAdapter, llm.GeminiAdapterInfo); err != nil {
		return fmt.Errorf("无法注册 gemini 适配器工厂: %v", err)
	}
	util.Debug("Gemini 提供者已注册")

	return nil
}
