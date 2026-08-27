package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/ai"
	"github.com/my-manus/my-manus-agent/internal/constant"
	"github.com/my-manus/my-manus-agent/internal/enum"
	"github.com/my-manus/my-manus-agent/internal/model"
	"github.com/my-manus/my-manus-agent/internal/prompt"
)

// browserAgent 浏览器自动化 Agent（对应 promptBrowserSystem / promptBrowserUserTask / promptPageStatus）
//
// 说明：完整实现需用 buildDomTree.js 生成可交互元素索引，支持 inputText/clickElement/switchTab 等动作。
// 当前实现已打通「goToUrl / extractContent / wait / done」的核心闭环，其余动作按 TODO 接入。
type browserAgent struct {
	model   ai.ChatClient
	prompts *prompt.Store
	cfg     *config.Config
}

func NewBrowserAgent(model ai.ChatClient, store *prompt.Store, cfg *config.Config) Agent {
	return &browserAgent{model: model, prompts: store, cfg: cfg}
}

func (a *browserAgent) Name() string { return enum.BrowserAgent.AgentName }

func (a *browserAgent) Execute(ctx context.Context, task string) (string, error) {
	system, _ := a.prompts.Get(constant.PromptBrowserSystem)
	userTaskTpl, _ := a.prompts.Get(constant.PromptBrowserUserTask)
	statusTpl, _ := a.prompts.Get(constant.PromptPageStatus)

	user := renderPrompt(userTaskTpl, map[string]string{"task": task})
	conversation := []ai.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	defer cancelBrowser()

	maxStep := 16 // 对应 generateNext.maxStep 上限
	for step := 1; step <= maxStep; step++ {
		raw, err := a.model.Chat(ctx, conversation)
		if err != nil {
			return "", err
		}
		out, err := parseAgentOutput(raw)
		if err != nil {
			return "", err
		}
		act := firstAction(out)
		if act == nil {
			return "", fmt.Errorf("浏览器输出缺少 action")
		}
		if act.Done != nil {
			return act.Done.Text, nil
		}

		// 浏览器 action 形如 {"goToUrl": {...}} 等，用 map 解析
		result, err := a.execBrowserAction(browserCtx, out.Action)
		if err != nil {
			result = fmt.Sprintf("动作执行失败: %v", err)
		}

		conversation = append(conversation, ai.Message{Role: "assistant", Content: raw})
		status := renderPrompt(statusTpl, map[string]string{
			"url":          a.currentURL(browserCtx),
			"pageListInfo": a.pageListInfo(browserCtx),
			"pageData":     a.pageData(browserCtx),
			"scrollData":   "",
			"stepData":     fmt.Sprintf("第 %d 步", step),
			"dateTime":     time.Now().Format("2006-01-02 15:04:05"),
		})
		conversation = append(conversation, ai.Message{Role: "user", Content: status + "\n\n[动作结果]\n" + result})
	}
	return "", fmt.Errorf("浏览器超过最大步数 %d", maxStep)
}

// execBrowserAction 执行浏览器动作（用 map 兼容 promptBrowserSystem 定义的动作集）
func (a *browserAgent) execBrowserAction(ctx context.Context, actions []model.ActionItem) (string, error) {
	var sb strings.Builder
	for _, item := range actions {
		// 将 ActionItem 序列化回 map 便于按动作名分发
		b, _ := json.Marshal(item)
		var m map[string]json.RawMessage
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		switch {
		case m["goToUrl"] != nil:
			var v struct {
				URL string `json:"url"`
			}
			_ = json.Unmarshal(m["goToUrl"], &v)
			if err := chromedp.Run(ctx, chromedp.Navigate(v.URL)); err != nil {
				return sb.String(), err
			}
			sb.WriteString("已导航到 " + v.URL + "\n")
		case m["extractContent"] != nil:
			var text string
			if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &text)); err != nil {
				return sb.String(), err
			}
			sb.WriteString("页面内容:\n" + text + "\n")
		case m["wait"] != nil:
			time.Sleep(2 * time.Second)
			sb.WriteString("已等待 2s\n")
		case m["done"] != nil:
			var v struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(m["done"], &v)
			return v.Text, nil
		default:
			sb.WriteString("动作暂未支持，已跳过\n")
		}
	}
	return sb.String(), nil
}

func (a *browserAgent) currentURL(ctx context.Context) string {
	var u string
	_ = chromedp.Run(ctx, chromedp.Location(&u))
	return u
}

func (a *browserAgent) pageData(ctx context.Context) string {
	var text string
	_ = chromedp.Run(ctx, chromedp.TextContent("body", &text, chromedp.ByQuery))
	return text
}

func (a *browserAgent) pageListInfo(ctx context.Context) string {
	// 简化：单标签页
	return "[0] " + a.currentURL(ctx)
}
