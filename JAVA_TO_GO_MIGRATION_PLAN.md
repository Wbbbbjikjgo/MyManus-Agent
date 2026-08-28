# MyManus Java → Go 一比一迁移方案

> 目标：将 `C:\agent\my-manus\my-manus` (Java + Vue) 后端全部用 Go 重写，前端 `my-manus-vue/` 保持不变。

---

## 一、项目现状分析

### 1.1 Java 项目扫描结果

| 维度 | 现状 |
|------|------|
| 框架 | Spring Boot 3.4.3 |
| Java 文件 | **仅 5 个**（入口类、配置类、枚举、常量、工具类） |
| Controller | **无** |
| Service | **无** |
| DAO/Repository | **无** |
| Entity/Model | **无**（仅有内部类 BaseModelConfig） |
| 数据库 | **无** |
| 安全认证 | **无** |
| 通信方式 | **纯 WebSocket STOMP**，无 REST API |

### 1.2 前端接口契约（不容变更）

| 项目 | 值 |
|------|-----|
| WebSocket URL | `ws://localhost:18081/bs-dialog-websocket` |
| 发送目标 | `/app/enhanced-dialog` |
| 订阅目标 | `/user/queue/dialog` |

**发送消息格式 (前端 → 后端)**：
```json
{"type": "user", "text": "用户输入的消息"}
```

**接收消息格式 (后端 → 前端)**：
```json
{
  "type": "server",
  "text": "回复文本",
  "imageUrl": "可选图片URL",
  "fileUrl": "可选文件URL",
  "meta": {"serverStatusHint": 0}
}
```
- `serverStatusHint`: `0` = 允许输入，`1` = 禁用输入（服务器忙）

---

## 二、技术栈对比

| 类别 | Java 技术栈 | Go 替代方案 | 选择理由 |
|------|------------|-------------|----------|
| **Web 框架** | Spring Boot 3.4.3 | **Gin v1.10** | 最流行、性能好、社区成熟 |
| **WebSocket** | spring-boot-starter-websocket (STOMP) | **gorilla/websocket** + 自实现 STOMP 子协议 | STOMP 协议简单，自实现完全可控 |
| **AI 应用框架** | Spring AI 抽象 | **Eino** (cloudwego/eino) | LLM 应用开发框架：ChatModel/ChatTemplate/Tool/Graph，是 Spring AI 的 Go 对应物 |
| **AI 客户端** | spring-ai-openai 1.0.0 | **Eino OpenAI ChatModel** (`eino-ext`) | 与 Spring AI 对应，支持自定义 BaseURL/流式/工具调用/结构化输出 |
| **MCP 客户端** | spring-ai-starter-mcp-client | **Eino MCP 组件** (`eino-ext`) 或 mcp-go | Eino 原生支持 MCP 工具，与 Agent 编排无缝集成 |
| **配置管理** | @ConfigurationProperties + application.yml | **Viper** | 支持 YAML、环境变量，与 Spring 风格相近 |
| **JSON 处理** | Jackson (Spring 内置) | **encoding/json** (标准库) | 标准库即可满足需求 |
| **HTML 解析** | Jsoup 1.17.2 | **goquery** | jQuery 风格选择器，与 Jsoup 最接近 |
| **HTML→Markdown** | flexmark-html2md-converter | **html-to-markdown** (JohannesKaufmann) | 功能完整 |
| **浏览器自动化** | Playwright 1.51.0 | **chromedp** 或 **rod** | chromedp 更底层，rod 更高级 |
| **文档解析** | Apache Tika 3.1.0 | **go-tika** 或直接调用 Tika Server | 如仅需基础解析可用标准库 |
| **工具类** | Hutool 5.8.36 | Go 标准库 + `lo`/`samber` 等 | Go 标准库功能已很丰富 |
| **ORM** | 无 | **GORM** (为将来扩展预留) | 当前无数据库需求 |
| **日志** | SLF4J (Spring 内置) | **zap** (uber-go) | 高性能结构化日志 |
| **构建** | Maven | **Go Modules** (`go mod`) | Go 原生构建系统 |
| **WebSocket Broker** | Spring STOMP Broker | **自实现 STOMP 层** | 仅需支持有限的 STOMP 帧类型 |

---

> **Eino 集成策略（新增）**：采用「组件级 + 编排级混合」——用 Eino 的 `ChatModel` / `ChatTemplate` 替代自研 AI 客户端，用 Eino `Graph` 做 Agent 调度骨架；但 13 个 prompt、`schemaBaseReAct.json`、`AgentOutput` 输出契约、`serverStatusHint` 机制全部 1:1 保留，不套用 Eino 内置 ReAct 消息流，确保与 Java 业务逻辑一致。

---

## 三、Go 项目目录结构（一比一对齐）

```
C:\agent\my-manus\MyManus-Agent\
├── my-manus-vue/                    # 【已有，不动】Vue 3 前端
├── go.mod                           # Go 模块定义
├── go.sum                           # 依赖锁定
├── main.go                          # ← MyManusApplication.java 入口
├── config/
│   ├── config.go                    # ← application.yml 配置结构体
│   └── model_config.go             # ← ModelConfig.java (ChatModel Bean 工厂)
├── internal/
│   ├── constant/
│   │   └── constant.go             # ← Constant.java (常量 + 提示词名称)
│   ├── enum/
│   │   └── agent_type.go           # ← AgentTypeEnum.java
│   ├── model/
│   │   └── message.go              # WebSocket 消息结构体
│   ├── util/
│   │   └── json_finder.go          # ← JsonFinder.java
│   ├── agent/
│   │   ├── base.go                 # Agent 基础接口定义
│   │   ├── graph.go                # ★ Eino Graph 编排（Planning → 子 Agent 调度）
│   │   ├── planning.go             # ReActPlanningAgent
│   │   ├── browser.go              # BrowserAgent
│   │   ├── table.go                # TableAgent
│   │   ├── chart.go                # ChartAgent
│   │   ├── htmldoc.go              # HtmlDocAgent
│   │   └── amap.go                 # AMAPAgent
│   ├── ai/
│   │   ├── model.go                # ★ Eino ChatModel 工厂（OpenAI 兼容 → ark）
│   │   └── prompt.go               # ★ Eino ChatTemplate 封装 + 提示词加载
│   ├── mcp/
│   │   └── client.go               # MCP 客户端（可选走 Eino MCP 组件）
│   ├── browser/
│   │   └── automation.go           # Playwright/chromedp 浏览器自动化
│   └── handler/
│       └── websocket_handler.go    # WebSocket + STOMP 处理 (核心!)
├── resources/
│   ├── application.yaml            # ← application.yml (Go 版本)
│   ├── schema/
│   │   └── schemaBaseReAct.json    # ← 直接复制
│   ├── prompt/                     # ← 13个提示词模板，直接复制
│   │   ├── promptAmapSystem.txt
│   │   ├── promptBrowserSystem.txt
│   │   ├── promptBrowserUserTask.txt
│   │   ├── promptChart.txt
│   │   ├── promptExtraPageContent.txt
│   │   ├── promptHtmlDoc.txt
│   │   ├── promptPageStatus.txt
│   │   ├── promptPlanningStatus.txt
│   │   ├── promptPlanningSystem.txt
│   │   ├── promptPlanningTaskMerging.txt
│   │   ├── promptPlanningUserTask.txt
│   │   ├── promptTable.txt
│   │   └── promptVisionExtract.txt
│   └── js/
│       └── buildDomTree.js          # ← 直接复制
└── file/                            # ← 文件存储目录 (对应 F:\tmp\my-manus\)
```

---

## 四、Java → Go 逐文件映射

### 4.1 入口类

| Java 文件 | Go 文件 | 说明 |
|-----------|---------|------|
| `MyManusApplication.java` | `main.go` | 启动 Gin 服务器、初始化 WebSocket、加载配置 |

**Java**:
```java
@SpringBootApplication
public class MyManusApplication {
    public static void main(String[] args) {
        SpringApplication.run(MyManusApplication.class, args);
    }
}
```

**Go** (`main.go`):
```go
func main() {
    cfg := config.Load()
    router := gin.Default()
    // 注册 WebSocket + STOMP 路由
    router.GET("/bs-dialog-websocket", handler.WebSocketHandler(cfg))
    // 静态文件服务
    router.Static("/file", cfg.File.Base)
    router.Run(":" + cfg.Server.Port) // 18081
}
```

### 4.2 配置类

| Java | Go | 说明 |
|------|-----|------|
| `ModelConfig.java` | `config/model_config.go` | ChatModel Bean → Go 工厂函数 |
| `application.yml` | `resources/application.yaml` | 配置结构体映射 |

**Java ModelConfig → Go 对应**:

```go
// 对应 Java 内部类 BaseModelConfig
type BaseModelConfig struct {
    BaseURL        string `mapstructure:"base-url"`
    APIKey         string `mapstructure:"api-key"`
    ModelName      string `mapstructure:"model-name"`
    CompletionsPath string `mapstructure:"completions-path"`
    Stream         bool   `mapstructure:"stream"`
}

// 对应 Java 的 mainAgentModel() + planModel() Bean → Eino ChatModel 工厂
// 端点 = base-url + completions-path，与 Java ModelConfig 的拼接逻辑保持一致
import openai "github.com/cloudwego/eino-ext/components/model/openai"

func NewMainAgentChatModel(ctx context.Context, cfg BaseModelConfig) (*openai.ChatModel, error) {
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        BaseURL: cfg.BaseURL, // 与 completions-path 拼接规则按 Java ModelConfig 原逻辑
        APIKey:  cfg.APIKey,
        Model:   cfg.ModelName,
        // Stream / ResponseFormat 等字段以 pin 定的 eino-ext 版本为准（示意）
    })
}
func NewPlanChatModel(ctx context.Context, cfg BaseModelConfig) (*openai.ChatModel, error) {
    // 同上，Model 使用 deepseek-r1
}
```

**YAML 配置 100% 保持一致**：
```yaml
server:
  port: 18081

agent-model:
  base-url: https://ark.cn-beijing.volces.com/api/
  api-key: ${VOLCES_API_KEY}
  model-name: deepseek-v3-250324
  completions-path: /v3/chat/completions
  stream: false

plan-model:
  base-url: https://ark.cn-beijing.volces.com/api/
  api-key: ${VOLCES_API_KEY}
  model-name: deepseek-r1-250120
  completions-path: /v3/chat/completions

file:
  base: F:\tmp\my-manus\
  domain: http://localhost:18081
```

### 4.3 常量 + 枚举

| Java | Go | 说明 |
|------|-----|------|
| `Constant.java` | `internal/constant/constant.go` | 字符串常量 | 
| `AgentTypeEnum.java` | `internal/enum/agent_type.go` | Agent 类型枚举 |

**Java Enum → Go (iota + struct)**：
```go
type AgentType struct {
    AgentName string
    Desc      string
}

var (
    AgentReActPlanning = AgentType{"reActPlanningAgent", "任务规划智能体"}
    AgentBrowser       = AgentType{"BrowserAgent", "浏览器Agent..."}
    AgentTable         = AgentType{"TableAgent", "表格绘制Agent"}
    AgentChart         = AgentType{"ChartAgent", "统计图绘制Agent"}
    AgentHtmlDoc       = AgentType{"HtmlDocAgent", "网页内容生成Agent"}
    AgentAMAP          = AgentType{"AMAPAgent", "高德地图工具集Agent"}
)

var agentTypes = []AgentType{...}

func AgentNameOf(name string) *AgentType { ... } // 对应 agentNameOf()
```

### 4.4 工具类

| Java | Go | 说明 |
|------|-----|------|
| `JsonFinder.java` | `internal/util/json_finder.go` | JSON 提取工具 |

### 4.5 资源文件（直接复制，无需转换）

| 文件 | 处理方式 |
|------|----------|
| `resources/schema/schemaBaseReAct.json` | 直接复制 |
| `resources/prompt/*.txt` (13个) | 直接复制 |
| `resources/js/buildDomTree.js` | 直接复制 |

---

## 五、WebSocket STOMP 接口——这是关键，必须100%兼容

### 5.1 STOMP 协议处理

Spring STOMP Broker 的核心功能需要在 Go 中实现：

| STOMP 帧 | 作用 | Go 实现 |
|----------|------|---------|
| `CONNECT` | 客户端连接 | 解析 headers，返回 `CONNECTED` |
| `SUBSCRIBE` | 订阅 `/user/queue/dialog` | 注册订阅关系 |
| `SEND` | 发送到 `/app/enhanced-dialog` | 路由到消息处理器 |
| `MESSAGE` | 服务端推送消息 | 发送到已订阅客户端 |
| `DISCONNECT` | 断开连接 | 清理订阅关系 |

### 5.2 消息流

```
前端(Dialog.vue)                          Go 后端
    |                                        |
    |--- CONNECT (ws://localhost:18081/bs-dialog-websocket) -->|
    |<-- CONNECTED -------------------------------------------|
    |                                        |
    |--- SUBSCRIBE /user/queue/dialog ----------------------->|
    |                                        |
    |--- SEND /app/enhanced-dialog -------------------------->|
    |    {"type":"user","text":"查询天气"}      |
    |                                        |--- Agent 处理 --->|
    |                                        |   (Planning → Browser/Chart/etc.)
    |<-- MESSAGE /user/queue/dialog --------------------------|
    |    {"type":"server","text":"...","meta":{"serverStatusHint":1}} (忙)
    |                                        |
    |<-- MESSAGE /user/queue/dialog --------------------------|
    |    {"type":"server","text":"结果...","meta":{"serverStatusHint":0}} (完成)
```

### 5.3 关键实现要点

```go
// handler/websocket_handler.go 核心逻辑

func WebSocketHandler(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)
        client := NewSTOMPClient(conn)
        
        go func() {
            for frame := range client.ReadFrames() {
                switch frame.Command {
                case "CONNECT":
                    client.SendCONNECTED()
                case "SUBSCRIBE":
                    client.RegisterSubscription(frame.Destination)
                case "SEND":
                    if frame.Destination == "/app/enhanced-dialog" {
                        go handleUserMessage(client, frame.Body)
                    }
                case "DISCONNECT":
                    client.Close()
                    return
                }
            }
        }()
    }
}

func handleUserMessage(client *STOMPClient, body []byte) {
    var msg Message
    json.Unmarshal(body, &msg)
    
    // 发送"忙"状态
    client.SendToSubscriber("/user/queue/dialog", Message{
        Type: "server",
        Meta: &Meta{ServerStatusHint: 1},
    })
    
    // Agent 处理...
    result := agent.Process(msg.Text)
    
    // 发送结果
    client.SendToSubscriber("/user/queue/dialog", Message{
        Type: "server",
        Text: result,
        Meta: &Meta{ServerStatusHint: 0},
    })
}
```

---

## 六、Agent 架构（Java 中的 6 种 Agent 设计需在 Go 中实现）

### 6.1 Agent 接口定义

```go
// internal/agent/base.go
type Agent interface {
    Name() string
    Execute(task string, ctx context.Context) (string, error)
}
```

> **编排层（★新增，Eino Graph）**：`Agent` 接口仍是各子 Agent 的一比一契约，
> 但「PlanningAgent 协调其他 Agent」的调度器改用 Eino `Graph` 建模——
> 节点 = 各 Agent，边 = 任务分发与结果归并。prompt/schema/输出契约保持 1:1，
> **不套用 Eino 内置 ReAct 消息流**，避免破坏与 Java 自定义规划循环（`promptPlanningSystem.txt` + `schemaBaseReAct.json`）的一致性。

```go
// internal/agent/graph.go（示意）
import (
    "github.com/cloudwego/eino/flow/graph"
    "github.com/cloudwego/eino/schema"
)

func buildAgentGraph(ctx context.Context) (*graph.Graph[string, *schema.Message], error) {
    g := graph.NewGraph[string, *schema.Message]()
    g.AddNode("planning", planningNode) // 规划 + 依据 AgentOutput.action 选择下一步
    g.AddNode("browser", browserNode)
    g.AddNode("table", tableNode)
    g.AddNode("chart", chartNode)
    g.AddNode("htmldoc", htmldocNode)
    g.AddNode("amap", amapNode)
    _ = g.AddEdge(graph.START, "planning")
    _ = g.AddBranch("planning", routeToNextAgent) // 依据 AgentOutput 返回下一节点名
    return g.Compile(ctx)
}
```

### 6.2 各 Agent 职责

| Agent | agentName | 职责 | 依赖 |
|-------|-----------|------|------|
| PlanningAgent | `reActPlanningAgent` | 任务分解、子任务调度、结果合并 | 主 AI 模型 |
| BrowserAgent | `BrowserAgent` | 浏览器自动化操作、网页信息提取 | chromedp/rod |
| TableAgent | `TableAgent` | 基于上下文数据生成表格 | 主 AI 模型 |
| ChartAgent | `ChartAgent` | 基于上下文数据生成统计图 | 主 AI 模型 |
| HtmlDocAgent | `HtmlDocAgent` | 生成网页内容（HTML文档） | 主 AI 模型 |
| AMAPAgent | `AMAPAgent` | 地图相关（路线规划、地理编码） | 高德 API |

### 6.3 AgentOutput Schema（JSON 格式，Go struct 定义）

```go
type AgentOutput struct {
    CurrentState CurrentState  `json:"current_state"`
    Action       []ActionItem  `json:"action"`
}

type CurrentState struct {
    EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
    Memory                 string `json:"memory"`
    Thinking               string `json:"thinking"`
}

type ActionItem struct {
    Done *DoneAction `json:"done,omitempty"`
    // 可扩展其他 action 类型
}

type DoneAction struct {
    Success bool   `json:"success"`
    Text    string `json:"text"`
}
```

---

## 七、Go 依赖清单 (go.mod)

```
module github.com/my-manus/my-manus-agent

go 1.22

require (
    github.com/cloudwego/eino v0.x     // ★ Eino 核心框架 (ChatModel/ChatTemplate/Tool/Graph)
    github.com/cloudwego/eino-ext v0.x // ★ Eino 扩展 (OpenAI ChatModel / MCP)
    github.com/gin-gonic/gin v1.10.0          // Web 框架
    github.com/gorilla/websocket v1.5.3       // WebSocket
    github.com/spf13/viper v1.19.0            // 配置管理
    github.com/PuerkitoBio/goquery v1.9.2     // HTML 解析 (替代 Jsoup)
    github.com/JohannesKaufmann/html-to-markdown v2.1.0 // HTML→MD
    github.com/chromedp/chromedp v0.10.0      // 浏览器自动化 (替代 Playwright)
    go.uber.org/zap v1.27.0                   // 结构化日志
    gorm.io/gorm v1.25.12                     // ORM (预留)
)
```

> Eino 系列版本以 `go get github.com/cloudwego/eino@latest` / `go get github.com/cloudwego/eino-ext@latest` 实际拉取的最新稳定版为准。

---

## 八、实施步骤

### 阶段一：基础骨架 (1:1 当前 5 个 Java 文件)

1. 初始化 Go module
2. 创建 `main.go`（入口）
3. 创建 `config/config.go` + `config/model_config.go`（配置 + 模型工厂）
4. 创建 `internal/constant/constant.go`（常量）
5. 创建 `internal/enum/agent_type.go`（枚举）
6. 创建 `internal/util/json_finder.go`（工具）
7. 复制所有 `resources/` 文件

### 阶段二：WebSocket STOMP 通信

8. 实现 `internal/model/message.go`（消息结构体）
9. 实现 STOMP 协议层（CONNECT/SUBSCRIBE/SEND/MESSAGE/DISCONNECT）
10. 实现 `internal/handler/websocket_handler.go`（WebSocket 处理器）
11. 实现消息转发逻辑：`/app/enhanced-dialog` → 处理 → `/user/queue/dialog`

### 阶段三：AI 客户端（Eino 组件层）

12. 引入 `github.com/cloudwego/eino` + `eino-ext`，用 `openai.NewChatModel` 实现 `internal/ai/model.go`（BaseURL/APIKey/ModelName/ResponseFormat 映射自 application.yaml）
13. 用 Eino `ChatTemplate` 封装 13 个提示词模板加载（`internal/ai/prompt.go`）
14. 实现 AgentOutput schema 解析（保持 `schemaBaseReAct.json` 与 Java 完全一致）

### 阶段四：Agent 实现

15. 实现 `internal/agent/base.go`（Agent 接口）
16. 实现 `internal/agent/planning.go`（任务规划智能体）
17. 实现 `internal/agent/browser.go`（浏览器智能体）
18. 实现 `internal/agent/table.go`（表格智能体）
19. 实现 `internal/agent/chart.go`（图表智能体）
20. 实现 `internal/agent/htmldoc.go`（HTML文档智能体）
21. 实现 `internal/agent/amap.go`（地图智能体）
22. 用 Eino `Graph` 实现 Agent 调度器（PlanningAgent 协调其他 Agent，`internal/agent/graph.go`）

### 阶段五：集成测试

23. 启动 Go 后端，验证端口 18081
24. 用前端连接 WebSocket，验证通信正常
25. 端到端测试：发送消息 → Agent 处理 → 返回结果

---

## 九、接口清单（前端依赖的全部接口）

| # | 接口 | 方式 | 说明 | 优先级 |
|---|------|------|------|--------|
| 1 | `ws://localhost:18081/bs-dialog-websocket` | WebSocket | STOMP 连接端点 | **P0** |
| 2 | STOMP `/app/enhanced-dialog` | SEND | 接收用户消息 | **P0** |
| 3 | STOMP `/user/queue/dialog` | MESSAGE | 推送回复消息 | **P0** |
| 4 | `GET /file/*` | HTTP | 静态文件服务（图片/文件） | P1 |

---

## 十、注意事项

1. **`serverStatusHint` 机制必须保留**：Agent 处理消息期间设为 1（禁用输入），完成后设为 0（启用输入），前端依赖此机制控制 UX。

2. **文件路径跨平台**：Java 中 `F:\tmp\my-manus\` 是 Windows 路径，Go 中需使用 `filepath` 包处理跨平台路径。

3. **环境变量**：`${VOLCES_API_KEY}` 在 Viper 中通过 `${VOLCES_API_KEY}` 或 `os.Getenv("VOLCES_API_KEY")` 读取。

4. **MCP 客户端**：如果 MCP 集成暂时不需要，可以先注释/跳过，不影响核心聊天功能。

5. **提示词模板**：13 个 `.txt` 文件直接复制，Go 中使用 `embed` 或者 `os.ReadFile` 加载。

6. **前端不需要任何改动**：两个 Dialog.vue 100% 一致，无需修改。

---

> **总结**：当前 Java 项目是一个骨架级别的项目（5 个 Java 文件，无业务逻辑），迁移到 Go 的实际工作量主要在**构建 Agent 执行引擎**（引入 Eino 后，模型组件与编排骨架大幅减负）和**自实现 STOMP 协议层**。前端接口契约已经明确，只需严格遵守消息格式即可保证兼容。
