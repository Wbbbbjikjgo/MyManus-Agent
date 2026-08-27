package agent

import (
	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/ai"
	"github.com/my-manus/my-manus-agent/internal/constant"
	"github.com/my-manus/my-manus-agent/internal/enum"
	"github.com/my-manus/my-manus-agent/internal/prompt"
)

// NewTableAgent 表格绘制 Agent（对应 promptTable）
func NewTableAgent(model ai.ChatClient, store *prompt.Store, cfg *config.Config) Agent {
	return &htmlAgent{name: enum.TableAgent.AgentName, model: model, prompts: store, cfg: cfg, promptName: constant.PromptTable}
}

// NewChartAgent 统计图绘制 Agent（对应 promptChart）
func NewChartAgent(model ai.ChatClient, store *prompt.Store, cfg *config.Config) Agent {
	return &htmlAgent{name: enum.ChartAgent.AgentName, model: model, prompts: store, cfg: cfg, promptName: constant.PromptChart}
}

// NewHtmlDocAgent 网页内容生成 Agent（对应 promptHtmlDoc）
func NewHtmlDocAgent(model ai.ChatClient, store *prompt.Store, cfg *config.Config) Agent {
	return &htmlAgent{name: enum.HtmlDocAgent.AgentName, model: model, prompts: store, cfg: cfg, promptName: constant.PromptHtmlDoc}
}
