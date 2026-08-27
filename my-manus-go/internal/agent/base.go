package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/ai"
	"github.com/my-manus/my-manus-agent/internal/model"
	"github.com/my-manus/my-manus-agent/internal/util"
)

// Agent 对应 Java AgentTypeEnum 中各智能体（除规划 Agent，它由 Engine 承担）
type Agent interface {
	Name() string
	Execute(ctx context.Context, task string) (string, error)
}

// parseAgentOutput 从 LLM 文本中提取并解析 AgentOutput（复刻 Java 的 JsonFinder + Jackson 流程）
func parseAgentOutput(raw string) (*model.AgentOutput, error) {
	js, ok := util.FindFirstJson(raw)
	if !ok {
		return nil, fmt.Errorf("输出中未找到合法 JSON")
	}
	var out model.AgentOutput
	if err := json.Unmarshal([]byte(js), &out); err != nil {
		return nil, fmt.Errorf("解析 AgentOutput: %w", err)
	}
	return &out, nil
}

// firstAction 返回第一个有效 action（prompt 规定每次仅 1 个）
func firstAction(out *model.AgentOutput) *model.ActionItem {
	for i := range out.Action {
		a := &out.Action[i]
		if a.GenerateNext != nil || a.Done != nil {
			return a
		}
	}
	return nil
}

// saveHTML 将生成的 HTML 落到 file.base，返回 file.domain 下的访问 URL
func saveHTML(cfg *config.Config, agentName, content string) (string, error) {
	if err := os.MkdirAll(cfg.File.Base, 0o755); err != nil {
		return "", err
	}
	fname := fmt.Sprintf("%s_%d.html", strings.ToLower(agentName), time.Now().UnixNano())
	if err := os.WriteFile(filepath.Join(cfg.File.Base, fname), []byte(content), 0o644); err != nil {
		return "", err
	}
	return strings.TrimRight(cfg.File.Domain, "/") + "/file/" + fname, nil
}

// stripMarkdownFence 移除 ```html 等代码块标记（prompt 要求输出纯 HTML）
func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```html")
	s = strings.TrimPrefix(s, "```HTML")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// htmlAgent 是 Table / Chart / HtmlDoc 三个“纯 LLM 生成 HTML”Agent 的公共实现
type htmlAgent struct {
	name       string
	model      ai.ChatClient
	prompts    promptStore
	cfg        *config.Config
	promptName string
}

// promptStore 缩小依赖面，便于注入（实现即 *prompt.Store）
type promptStore interface {
	Get(name string) (string, error)
}

func (a *htmlAgent) Name() string { return a.name }

func (a *htmlAgent) Execute(ctx context.Context, task string) (string, error) {
	tpl, err := a.prompts.Get(a.promptName)
	if err != nil {
		return "", err
	}
	user := renderPrompt(tpl, map[string]string{"task": task})
	html, err := a.model.Chat(ctx, []ai.Message{{Role: "user", Content: user}})
	if err != nil {
		return "", err
	}
	return saveHTML(a.cfg, a.name, stripMarkdownFence(html))
}
