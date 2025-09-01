package react

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"ai-ops/internal/llm"
	"ai-ops/internal/tools"
)

const MaxSteps = 15

// Agent ReAct智能体
type Agent struct {
	client      llm.ModelAdapter
	toolManager tools.ToolManager
	toolDefs    []tools.ToolDefinition
}

// NewAgent 创建新的ReAct智能体
func NewAgent(client llm.ModelAdapter, toolManager tools.ToolManager) *Agent {
	return &Agent{
		client:      client,
		toolManager: toolManager,
		toolDefs:    toolManager.GetToolDefinitions(),
	}
}

// Execute 执行ReAct任务
func (a *Agent) Execute(ctx context.Context, task string, debug bool) (string, error) {
	var steps []Step
	messages := []llm.Message{
		{Role: "system", Content: a.getSystemPrompt()},
		{Role: "user", Content: fmt.Sprintf("任务: %s", task)},
	}

	for stepNum := 1; stepNum <= MaxSteps; stepNum++ {
		// 发送消息给AI
		resp, err := a.client.SendMessage(ctx, messages, a.toolDefs)
		if err != nil {
			return "", fmt.Errorf("步骤 %d AI响应失败: %w", stepNum, err)
		}

		// 解析AI响应
		step, finished, err := a.parseResponse(stepNum, *resp)
		if err != nil {
			return "", fmt.Errorf("步骤 %d 解析响应失败: %w", stepNum, err)
		}

		steps = append(steps, step)

		// 显示当前步骤（如果开启debug模式）
		if debug {
			a.printStep(step)
		}

		// 将AI响应添加到消息历史
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// 如果AI表示完成，返回最终答案
		if finished {
			return step.Observation, nil
		}

		// 如果有工具调用，执行工具并添加结果到消息
		if len(resp.ToolCalls) > 0 {
			toolResults, err := a.executeTools(ctx, resp.ToolCalls)
			if err != nil {
				return "", fmt.Errorf("步骤 %d 执行工具失败: %w", stepNum, err)
			}
			messages = append(messages, toolResults...)

			// 更新步骤的观察结果
			if len(toolResults) > 0 {
				step.Observation = toolResults[0].Content
				steps[len(steps)-1] = step
			}
		}
	}

	return "", fmt.Errorf("达到最大步数 %d，任务未完成", MaxSteps)
}

// parseResponse 解析AI响应，提取思考、行动和观察
func (a *Agent) parseResponse(stepNum int, resp llm.Response) (Step, bool, error) {
	step := Step{
		StepNumber: stepNum,
	}

	content := resp.Content

	// 提取思考过程
	thoughtPattern := regexp.MustCompile(`(?i)思考[:：]\s*(.+?)(?:\n|$)`)
	if match := thoughtPattern.FindStringSubmatch(content); len(match) > 1 {
		step.Thought = strings.TrimSpace(match[1])
	} else {
		// 如果没有明确的"思考："标记，使用整个内容作为思考
		step.Thought = content
	}

	// 检查是否完成任务
	finishPattern := regexp.MustCompile(`(?is)最终答案[:：]\s*(.*)`)
	if match := finishPattern.FindStringSubmatch(content); len(match) > 1 {
		step.Action = "完成任务"
		step.Observation = strings.TrimSpace(match[1])
		return step, true, nil
	}
	
	// 如果没有工具调用且没有"最终答案"标记，可能就是最终回答
	if len(resp.ToolCalls) == 0 && (resp.FinishReason == "stop" || resp.FinishReason == "STOP") {
		step.Action = "完成任务"
		// 确保观察结果不为空
		if strings.TrimSpace(content) != "" {
			step.Observation = content
			return step, true, nil
		}
	}

	// 如果有工具调用
	if len(resp.ToolCalls) > 0 {
		toolCall := resp.ToolCalls[0] // 简化：只处理第一个工具调用
		step.Action = fmt.Sprintf("调用工具: %s", toolCall.Name)
		step.Tool = toolCall.Name
		step.Args = toolCall.Arguments
	} else {
		step.Action = "继续思考"
	}

	return step, false, nil
}

// executeTools 执行工具调用
func (a *Agent) executeTools(ctx context.Context, toolCalls []llm.ToolCall) ([]llm.Message, error) {
	var toolMessages []llm.Message

	for _, tc := range toolCalls {
		result, err := a.toolManager.ExecuteToolCall(ctx, tools.ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: tc.Arguments,
		})

		var content string
		if err != nil {
			content = fmt.Sprintf("工具执行错误: %v", err)
		} else {
			content = result
		}

		toolMessage := llm.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: tc.ID,
			Name:       tc.Name,
		}
		toolMessages = append(toolMessages, toolMessage)
	}

	return toolMessages, nil
}

// printStep 打印步骤信息（debug模式）
func (a *Agent) printStep(step Step) {
	fmt.Printf("\n=== 步骤 %d ===\n", step.StepNumber)
	fmt.Printf("思考: %s\n", step.Thought)
	fmt.Printf("行动: %s\n", step.Action)
	if step.Tool != "" {
		argsStr, _ := json.Marshal(step.Args)
		fmt.Printf("工具: %s(%s)\n", step.Tool, string(argsStr))
	}
	if step.Observation != "" {
		fmt.Printf("观察: %s\n", step.Observation)
	}
}

// getSystemPrompt 获取ReAct系统提示词
func (a *Agent) getSystemPrompt() string {
	toolDescriptions := ""
	for _, tool := range a.toolDefs {
		toolDescriptions += fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description)
	}

	return fmt.Sprintf(`你是一个ReAct (Reasoning and Acting)智能体。你需要通过思考和行动来完成用户的任务。

可用工具:
%s

工作模式:
1. 思考: 分析问题，制定下一步计划
2. 行动: 调用工具获取信息或执行操作
3. 观察: 分析工具返回的结果
4. 重复上述过程直到完成任务

响应格式:
- 每次响应都要包含"思考："来说明你的推理过程
- 需要调用工具时直接使用function calling
- 完成任务时，必须以"最终答案："开头，然后给出完整详细的结果

注意事项:
- 每次只专注一个具体的子任务
- 充分利用工具获取信息
- 基于观察结果调整策略
- 最终答案必须完整、详细且格式化良好`, toolDescriptions)
}
