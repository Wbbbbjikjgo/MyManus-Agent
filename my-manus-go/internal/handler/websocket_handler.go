package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/agent"
	"github.com/my-manus/my-manus-agent/internal/model"
	"github.com/my-manus/my-manus-agent/internal/stomp"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler 处理 STOMP over WebSocket（前端 Dialog.vue 契约）
type Handler struct {
	cfg    *config.Config
	engine *agent.Engine
}

func New(cfg *config.Config) (*Handler, error) {
	engine, err := agent.NewEngine(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	return &Handler{cfg: cfg, engine: engine}, nil
}

// Serve 是 gin 的 WebSocket 入口
func (h *Handler) Serve(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &stompClient{conn: conn, subs: map[string]string{}}
	client.run(h)
}

type stompClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
	subs map[string]string // subscription id -> destination
}

func (c *stompClient) run(h *Handler) {
	defer c.conn.Close()
	var buf []byte
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		buf = append(buf, data...)
		for {
			frame, rest, ok := extractFrame(buf)
			if !ok {
				break
			}
			buf = rest
			if !h.dispatch(c, frame) {
				return
			}
		}
	}
}

// extractFrame 从缓冲区取出首个完整 STOMP 帧（以 \0 结尾）
func extractFrame(buf []byte) (*stomp.Frame, []byte, bool) {
	idx := bytes.IndexByte(buf, 0)
	if idx < 0 {
		return nil, buf, false
	}
	frame, err := stomp.Parse(buf[:idx+1])
	if err != nil {
		log.Printf("跳过无效帧: %v", err)
		return nil, buf[idx+1:], true
	}
	return frame, buf[idx+1:], true
}

func (h *Handler) dispatch(c *stompClient, f *stomp.Frame) bool {
	switch f.Command {
	case "CONNECT":
		c.send(&stomp.Frame{Command: "CONNECTED", Headers: map[string]string{
			"version":    "1.2",
			"heart-beat": "0,0",
		}})
		return true
	case "SUBSCRIBE":
		c.mu.Lock()
		c.subs[f.Headers["id"]] = f.Headers["destination"]
		c.mu.Unlock()
		return true
	case "UNSUBSCRIBE":
		c.mu.Lock()
		delete(c.subs, f.Headers["id"])
		c.mu.Unlock()
		return true
	case "SEND":
		if f.Headers["destination"] == "/app/enhanced-dialog" {
			go h.handleUserMessage(c, f.Body)
		}
		return true
	case "DISCONNECT":
		return false
	default:
		return true
	}
}

func (h *Handler) handleUserMessage(c *stompClient, body []byte) {
	var msg model.Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return
	}
	if msg.Type != "user" || msg.Text == "" {
		return
	}

	// 先发“忙”状态（禁用输入）
	c.sendMessage(model.NewServerMessage("", 1))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	result, err := h.engine.Run(ctx, msg.Text)
	if err != nil {
		result = "处理失败: " + err.Error()
	}
	// 结果（放行输入）
	c.sendMessage(model.NewServerMessage(result, 0))
}

func (c *stompClient) sendMessage(msg model.Message) {
	data, _ := json.Marshal(msg)
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, dest := range c.subs {
		if dest == "/user/queue/dialog" {
			c.sendLocked(&stomp.Frame{
				Command: "MESSAGE",
				Headers: map[string]string{
					"destination":    dest,
					"subscription":   id,
					"content-type":   "application/json",
					"content-length": strconv.Itoa(len(data)),
				},
				Body: data,
			})
		}
	}
}

func (c *stompClient) send(f *stomp.Frame) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sendLocked(f)
}

func (c *stompClient) sendLocked(f *stomp.Frame) {
	if err := c.conn.WriteMessage(websocket.TextMessage, f.Encode()); err != nil {
		log.Printf("写帧失败: %v", err)
	}
}
