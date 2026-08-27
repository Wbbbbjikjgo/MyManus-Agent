package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/my-manus/my-manus-agent/config"
)

// Message 引擎内部使用的消息
type Message struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// ChatClient 聊天模型抽象，Agent 引擎只依赖此接口
type ChatClient interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}

// httpClient 直接调用 OpenAI 兼容的 /chat/completions（DeepSeek 等）
type httpClient struct {
	baseURL   string
	apiKey    string
	modelName string
	client    *http.Client
}

// NewChatModel 对应 Java ModelConfig.buildByConfig()。
// base-url + completions-path 拼成完整端点，例如 https://api.deepseek.com/v1/chat/completions
func NewChatModel(ctx context.Context, cfg config.BaseModelConfig) (ChatClient, error) {
	_ = ctx
	return &httpClient{
		baseURL:   resolveBaseURL(cfg),
		apiKey:    cfg.APIKey,
		modelName: cfg.ModelName,
		client:    &http.Client{Timeout: 10 * time.Minute},
	}, nil
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (c *httpClient) Chat(ctx context.Context, messages []Message) (string, error) {
	payload, err := json.Marshal(chatRequest{Model: c.modelName, Messages: messages, Stream: false})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("模型 HTTP %d: %s", resp.StatusCode, string(data))
	}

	var cr chatResponse
	if err := json.Unmarshal(data, &cr); err != nil {
		return "", fmt.Errorf("解析模型响应失败: %w", err)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("模型错误: %s (%s)", cr.Error.Message, cr.Error.Type)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("模型响应无 choices: %s", string(data))
	}
	return cr.Choices[0].Message.Content, nil
}

// resolveBaseURL 复刻 Java 的 baseUrl + completionsPath 拼接：
//
//	https://api.deepseek.com/v1  +  /chat/completions
//	= https://api.deepseek.com/v1/chat/completions
func resolveBaseURL(cfg config.BaseModelConfig) string {
	base := strings.TrimRight(cfg.BaseURL, "/")
	path := strings.Trim(cfg.CompletionsPath, "/")
	if path == "" {
		return base
	}
	return base + "/" + path
}
