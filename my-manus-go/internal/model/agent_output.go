package model

// AgentOutput 对应 schemaBaseReAct.json（规划/浏览器等 Agent 的统一结构化输出）
type AgentOutput struct {
	CurrentState CurrentState `json:"current_state"`
	Action       []ActionItem `json:"action"` // prompt 规定每次仅 1 个 action
}

type CurrentState struct {
	EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
	Memory                 string `json:"memory"`
	Thinking               string `json:"thinking"`
}

// ActionItem 每个元素形如 {"generateNext": {...}} 或 {"done": {...}}
type ActionItem struct {
	GenerateNext *GenerateNextAction `json:"generateNext,omitempty"`
	Done         *DoneAction         `json:"done,omitempty"`
}

type GenerateNextAction struct {
	Agent   string `json:"agent"`
	SubTask string `json:"subTask"`
	MaxStep int    `json:"maxStep"`
}

type DoneAction struct {
	Success bool   `json:"success"`
	Text    string `json:"text"`
}
