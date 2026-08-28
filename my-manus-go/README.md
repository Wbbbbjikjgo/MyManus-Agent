# my-manus-go

MyManus Java 后端的 Go 版实现（对应 `../my-manus` 的 Java 项目）。

## 目录结构

```
my-manus-go/
├── main.go                     # 入口（对应 MyManusApplication.java）
├── config/config.go            # 配置 + BaseModelConfig（对应 ModelConfig.java）
├── internal/
│   ├── constant/constant.go    # 常量 + 13 个 prompt 名称（对应 Constant.java）
│   ├── enum/agent_type.go      # 6 个 Agent 枚举（对应 AgentTypeEnum.java）
│   ├── util/json_finder.go     # 提取首个合法 JSON（对应 JsonFinder.java）
│   ├── model/                  # 消息 + AgentOutput 结构体
│   ├── ai/client.go            # DeepSeek OpenAI 兼容 HTTP 客户端
│   ├── prompt/prompt.go        # 提示词加载 + 占位符填充
│   ├── stomp/stomp.go          # STOMP 子协议编解码
│   ├── handler/websocket_handler.go  # WebSocket + STOMP 端点（前端契约）
│   └── agent/                  # 规划引擎 + 6 个 Agent
└── resources/                  # application.yaml + schema + 13 prompt + js
```

## 前置条件

- Go 1.22+
- API Key 已配置在 `resources/application.yaml`（DeepSeek）
- 浏览器 Agent 需要本机安装 Chrome/Chromium

## 构建与运行

```powershell
cd C:\agent\my-manus\MyManus-Agent\my-manus-go

# 拉取依赖（首次）
go mod tidy

# 编译
go build -o my-manus-go.exe .

# 运行（监听 18081）
.\my-manus-go.exe
```

## 测试

前端 `../my-manus/my-manus-vue` 无需改动，启动后它会连接 `ws://localhost:18081/bs-dialog-websocket`。

也可以用任意 STOMP 客户端测试：

```
CONNECT
accept-version:1.2

SUBSCRIBE
id:sub-0
destination:/user/queue/dialog

SEND
destination:/app/enhanced-dialog
content-type:application/json

{"type":"user","text":"查询今天北京的天气"}
```

## 与 Java 的差异（有意为之）

1. `file.base` 由 `F:\tmp\my-manus\` 改为相对路径 `./file/`，便于本地运行（见 `resources/application.yaml`）。
2. `BaseModelConfig.stream` 在 Java 中绑定但未使用，Go 版同样不实现流式。
3. 高德 AMAP 工具链、浏览器完整交互动作（inputText/clickElement/switchTab）留有接入点，见对应源码 TODO。

## Git

```powershell
git add .
git commit -m "feat: my-manus Java 后端 Go 实现（Eino + STOMP + ReAct 引擎）"
```
