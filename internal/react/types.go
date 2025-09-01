package react

// Step 表示ReAct执行的一个步骤
type Step struct {
	StepNumber int    `json:"step_number"`
	Thought    string `json:"thought"`    // 思考过程
	Action     string `json:"action"`     // 执行的动作
	Tool       string `json:"tool"`       // 使用的工具名称
	Args       map[string]any `json:"args"` // 工具参数
	Observation string `json:"observation"` // 观察结果
}

// ExecutionResult 表示整个执行过程的结果
type ExecutionResult struct {
	Task         string `json:"task"`          // 原始任务
	Steps        []Step `json:"steps"`         // 执行步骤
	FinalAnswer  string `json:"final_answer"`  // 最终答案
	TotalSteps   int    `json:"total_steps"`   // 总步数
	Success      bool   `json:"success"`       // 是否成功完成
}

// ActionType 动作类型
type ActionType string

const (
	ActionThink  ActionType = "think"  // 纯思考，不调用工具
	ActionTool   ActionType = "tool"   // 调用工具
	ActionFinish ActionType = "finish" // 完成任务
)