package main

import (
	"log"
	"strconv"

	"github.com/my-manus/my-manus-agent/config"
	"github.com/my-manus/my-manus-agent/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("resources/application.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	ws, err := handler.New(cfg)
	if err != nil {
		log.Fatalf("初始化 WebSocket 处理器失败: %v", err)
	}

	r := gin.Default()
	// 静态文件服务：对应 Java file.base / file.domain（图片、生成的 HTML 等）
	r.Static("/file", cfg.File.Base)
	// STOMP WebSocket 端点：前端 Dialog.vue 契约 ws://localhost:18081/bs-dialog-websocket
	r.GET("/bs-dialog-websocket", ws.Serve)

	addr := ":" + strconv.Itoa(cfg.Server.Port)
	log.Printf("my-manus-go 监听 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
