package plugins

import (
	"context"
	"fmt"

	"ai-ops/internal/util"
)

// EchoTool 回显工具实现
type EchoTool struct{}

func (e *EchoTool) Name() string        { return "echo" }
func (e *EchoTool) Description() string { return "回显输入的消息" }
func (e *EchoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "要回显的消息内容",
			},
		},
		"required": []string{"message"},
	}
}

func (e *EchoTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	message, ok := args["message"].(string)
	if !ok || message == "" {
		return "", util.NewError(util.ErrCodeInvalidParam, "缺少或无效的 message 参数")
	}

	util.Infow("执行回显工具", map[string]any{"message": message})
	return fmt.Sprintf("回显: %s", message), nil
}

// NewEchoTool 创建回显工具实例
func NewEchoTool() interface{} {
	return &EchoTool{}
}

// init 函数用于自动注册工具
func init() {
	// 延迟注册，避免循环导入
	go func() {
		// 这里需要一个更好的方式来注册插件
		// 暂时先注释掉，在plugin_loader中直接创建
	}()
}
