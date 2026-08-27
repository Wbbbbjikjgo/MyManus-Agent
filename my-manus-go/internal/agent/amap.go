package agent

import (
	"context"

	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/ai"
	"github.com/my-manus/my-manus-agent/internal/constant"
	"github.com/my-manus/my-manus-agent/internal/enum"
	"github.com/my-manus/my-manus-agent/internal/prompt"
)

// amapAgent 高德地图工具集 Agent。
// 完整版需接入高德 API 工具链（地理编码/路线规划等，可通过 MCP 或直接 HTTP 实现），
// 当前先以系统提示词驱动模型；工具链接入点见注释。
type amapAgent struct {
	model   ai.ChatClient
	prompts *prompt.Store
	cfg     *config.Config
}

func NewAMAPAgent(model ai.ChatClient, store *prompt.Store, cfg *config.Config) Agent {
	return &amapAgent{model: model, prompts: store, cfg: cfg}
}

func (a *amapAgent) Name() string { return enum.AMAPAgent.AgentName }

func (a *amapAgent) Execute(ctx context.Context, task string) (string, error) {
	tpl, err := a.prompts.Get(constant.PromptAmapSystem)
	if err != nil {
		return "", err
	}
	system := renderPrompt(tpl, map[string]string{"task": task})
	// TODO(高德工具链)：在此处注入高德 API 工具（geocode/route 等）并走 function-calling 循环。
	// 未配置工具时，模型按自身知识回答。
	return a.model.Chat(ctx, []ai.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: task},
	})
}
