// testclient 是一个最小 STOMP 客户端，用于端到端测试 my-manus-go。
// 用法：go run ./testclient [ws地址] [测试问题]
// 示例：go run ./testclient ws://localhost:18081/bs-dialog-websocket "把1月、2月、3月的销售额100、200、150做成表格"
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := "ws://localhost:18081/bs-dialog-websocket"
	if len(os.Args) > 1 {
		url = os.Args[1]
	}
	question := "把1月、2月、3月的销售额100、200、150做成表格"
	if len(os.Args) > 2 {
		question = os.Args[2]
	}

	fmt.Println("连接:", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println("❌ 连接失败:", err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Println("✅ WebSocket 已连接")

	// 1. CONNECT
	send(conn, "CONNECT\naccept-version:1.2\nhost:localhost\n\n\x00")
	// 2. SUBSCRIBE /user/queue/dialog
	send(conn, "SUBSCRIBE\nid:sub-0\ndestination:/user/queue/dialog\n\n\x00")

	// 3. SEND /app/enhanced-dialog
	body, _ := json.Marshal(map[string]string{"type": "user", "text": question})
	send(conn, "SEND\ndestination:/app/enhanced-dialog\ncontent-type:application/json\n\n"+string(body)+"\x00")
	fmt.Println("📤 已发送问题:", question)

	// 4. 读取服务端返回的帧
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("读取结束:", err)
				return
			}
			fmt.Println("\n===== 收到服务端帧 =====")
			fmt.Println(string(data))
		}
	}()

	select {
	case <-done:
		fmt.Println("\n✅ 测试完成")
	case <-time.After(3 * time.Minute):
		fmt.Println("\n⏱️ 超时（3 分钟），可能是模型响应慢")
	}
}

func send(conn *websocket.Conn, s string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(s)); err != nil {
		fmt.Println("❌ 发送失败:", err)
		os.Exit(1)
	}
}
