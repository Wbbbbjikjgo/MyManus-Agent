package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/ai"
	"github.com/my-manus/my-manus-agent/internal/constant"
	"github.com/my-manus/my-manus-agent/internal/enum"
	"github.com/my-manus/my-manus-agent/internal/prompt"
)

const maxPlanSteps = 30 // 规划循环安全上限

// Engine 复刻 ReAct 规划循环：PlanningAgent 拆解任务并调度各子 Agent
type Engine struct {
	planModel  ai.ChatClient
	agentModel ai.ChatClient
	prompts    *prompt.Store
	cfg        *config.Config
	agents     map[string]Agent
}

func NewEngine(ctx context.Context, cfg *config.Config) (*Engine, error) {
	plan, err := ai.NewChatModel(ctx, cfg.PlanModel)
	if err != nil {
		return nil, fmt.Errorf("初始化 planModel: %w", err)
	}
	mainAgent, err := ai.NewChatModel(ctx, cfg.AgentModel)
	if err != nil {
		return nil, fmt.Errorf("初始化 mainAgentModel: %w", err)
	}
	store, err := prompt.Load("resources/prompt")
	if err != nil {
		return nil, err
	}

	e := &Engine{
		planModel:  plan,
		agentModel: mainAgent,
		prompts:    store,
		cfg:        cfg,
		agents:     map[string]Agent{},
	}
	e.register(NewTableAgent(mainAgent, store, cfg))
	e.register(NewChartAgent(mainAgent, store, cfg))
	e.register(NewHtmlDocAgent(mainAgent, store, cfg))
	e.register(NewAMAPAgent(mainAgent, store, cfg))
	e.register(NewBrowserAgent(mainAgent, store, cfg))
	return e, nil
}

func (e *Engine) register(a Agent) { e.agents[a.Name()] = a }

type subTaskResult struct {
	Agent  string
	Task   string
	Result string
}

// Run 处理一条用户消息，返回最终结果
func (e *Engine) Run(ctx context.Context, userText string) (string, error) {
	planningSystem, _ := e.prompts.Get(constant.PromptPlanningSystem)
	planningUserTask, _ := e.prompts.Get(constant.PromptPlanningUserTask)
	planningStatus, _ := e.prompts.Get(constant.PromptPlanningStatus)

	conversation := []ai.Message{
		{Role: "system", Content: planningSystem},
		{Role: "user", Content: renderPrompt(planningUserTask, map[string]string{
			"task":      userText,
			"agentData": enum.AgentData(),
		})},
	}

	var chain []subTaskResult

	for step := 1; step <= maxPlanSteps; step++ {
		raw, err := e.planModel.Chat(ctx, conversation)
		if err != nil {
			return "", fmt.Errorf("调用规划模型: %w", err)
		}

		out, err := parseAgentOutput(raw)
		if err != nil {
			return "", err
		}

		act := firstAction(out)
		if act == nil {
			return "", fmt.Errorf("规划输出缺少 action")
		}

		if act.Done != nil {
			return act.Done.Text, nil
		}

		if act.GenerateNext != nil {
			gen := act.GenerateNext
			sub, ok := e.agents[gen.Agent]
			if !ok {
				return "", fmt.Errorf("未知 Agent: %s", gen.Agent)
			}
			result, err := sub.Execute(ctx, gen.SubTask)
			if err != nil {
				result = fmt.Sprintf("子任务执行失败: %v", err)
			}
			chain = append(chain, subTaskResult{Agent: gen.Agent, Task: gen.SubTask, Result: result})

			conversation = append(conversation, ai.Message{Role: "assistant", Content: raw})
			conversation = append(conversation, ai.Message{Role: "user", Content: renderPrompt(planningStatus, map[string]string{
				"subTaskChain": renderChain(chain),
				"latestResult": result,
				"stepData":     fmt.Sprintf("第 %d 步", step),
				"dateTime":     time.Now().Format("2006-01-02 15:04:05"),
			})})
			continue
		}

		return "", fmt.Errorf("未知 action 类型")
	}

	return "", fmt.Errorf("超出最大规划步数 %d", maxPlanSteps)
}

func renderChain(chain []subTaskResult) string {
	var sb strings.Builder
	for i, r := range chain {
		if i > 0 {
			sb.WriteString(" -> ")
		}
		sb.WriteString(fmt.Sprintf("[%s,Task:%s,Result:%s]", r.Agent, r.Task, r.Result))
	}
	return sb.String()
}

// renderPrompt 便捷封装
func renderPrompt(tpl string, vars map[string]string) string {
	return prompt.Render(tpl, vars)
}
