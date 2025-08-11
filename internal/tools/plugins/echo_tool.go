package plugins

import (
	"context"
	"fmt"

	"ai-ops/internal/common/errors"
	"ai-ops/internal/util"
)

// EchoTool 回显工具实现
type EchoTool struct{}

func (e *EchoTool) ID() string          { return "echo" }
func (e *EchoTool) Name() string        { return "echo" }
func (e *EchoTool) Type() string        { return "plugin" }
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
		return "", errors.NewError(errors.ErrCodeInvalidParam, "缺少或无效的 message 参数")
	}

	util.Infow("执行回显工具", map[string]any{"message": message})
	return fmt.Sprintf("蓝胖说: %s", message), nil
}

// NewEchoTool 创建回显工具实例
func NewEchoTool() interface{} {
	return &EchoTool{}
}
