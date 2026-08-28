# MyManus Java → Go 逐行对照表（基于真实源码扫描）

> 依据：真实扫描工作区 `my-manus\` 目录，逐文件阅读全部 5 个 Java 文件、`pom.xml`、`application.yml`、`schemaBaseReAct.json`、13 个 prompt、前端 `Dialog.vue`，并对全源码 grep `websocket/stomp/controller/service/MessageMapping`（零命中）。
>
> 本文档是**代码级权威对照**，修正并取代原 `JAVA_TO_GO_MIGRATION_PLAN.md` 中 §4.2 / §4.3 / §6.3 的推测性内容。

---

## 一、最重要的结论（先看这个）

1. **Java 后端 = 纯配置骨架。** 5 个 Java 文件里没有 Controller、没有 Service、没有 WebSocket、没有 STOMP 端点、没有任何 Agent 执行代码。
2. 因此「完美复刻 Java 业务逻辑」的真实含义是：
   - 1:1 复刻 **5 个琐碎文件**（入口 / 模型配置 / 常量 / 枚举 / JSON 工具）——工作量极小；
   - 1:1 保留 **前端契约**（`Dialog.vue`，无需改动）；
   - 1:1 复制 **prompt + schema**（直接拷贝）；
   - **在 Go 里从零构建**：WebSocket STOMP 端点 + ReAct Agent 执行引擎 + 6 个 Agent + 浏览器自动化 + AMAP + MCP。
3. 真正要「复刻」的“业务逻辑”**不是 Java 代码**，而是声明式地写在这三处：
   - `schemaBaseReAct.json` —— 规划循环的结构化输出契约；
   - `promptPlanningSystem.txt`（+ 其余 12 个 prompt）—— 每个 Agent 的行为规则；
   - `Dialog.vue` —— 前后端消息交互契约。
4. **这正是 Eino 价值最大的场景**：引擎是从零搭的，用 Eino 的 `ChatModel` + `ChatTemplate` + `Graph` 搭骨架，比手写省大量代码，同时不违背 1:1（因为 Java 里本就没有引擎可“对照”）。

---

## 二、相对原方案的 4+1 处修正

| # | 原方案写法 | 真实值（已核对源码） | 修正 |
|---|-----------|---------------------|------|
| 1 | `file.base: F:\tmp\my-manus\`（单反斜杠，**非法 YAML**） | `application.yml` 里是 `F:\\tmp\\my-manus\\`（转义后 = `F:\tmp\my-manus\`） | Go 配置映射时用原始字符串，别把双反斜杠当成字面量 |
| 2 | 枚举 desc 缩略（如“浏览器Agent...”） | `AgentTypeEnum.java` 是完整长句（见 §4.4） | 1:1 复刻必须用原文 |
| 3 | 端点拼接未说明 | `baseUrl + completionsPath` = `https://ark.cn-beijing.volces.com/api/` + `/v3/chat/completions` = `https://ark.cn-beijing.volces.com/api/v3/chat/completions` | Eino `BaseURL` 应取 `.../api/v3` 前缀（见 §4.2） |
| 4 | `BaseModelConfig.stream` 默认照搬 | Java 里 `stream` 字段**绑定了但 `buildByConfig()` 从未使用** | Go 不必实现流式，避免过度设计 |
| 5 | §6.3 `AgentOutput` 只定义了 `done` | 真实 action 有 **两种**：`generateNext` 与 `done`（见 §5.1） | 补充 `generateNext` |

---

## 三、前端契约实测（`Dialog.vue`）

前端用 `@stomp/stompjs`，契约如下（这是**不容变更**的硬约束）：

```js
// 连接
new Client({ brokerURL: 'ws://localhost:18081/bs-dialog-websocket' });

// 订阅
stompClient.subscribe('/user/queue/dialog', handleMessage);

// 发送
stompClient.publish({
  destination: '/app/enhanced-dialog',
  body: JSON.stringify({ type: 'user', text: this.newMessage })
});

// 接收处理
handleMessage(playload) {
  const message = JSON.parse(playload.body);
  if (message.text) this.messages.push(message);      // 只有 text 非空才渲染
  if (message.meta) {
    if (message.meta.serverStatusHint == 0) this.disableInput = false;
    else if (message.meta.serverStatusHint == 1) this.disableInput = true;
  }
}
```

**结论**：
- 后端必须实现 STOMP 子协议：`CONNECT` / `SUBSCRIBE /user/queue/dialog` / `SEND /app/enhanced-dialog` / `MESSAGE` / `DISCONNECT`。
- 服务端推送消息结构：`{ type:"server", text, imageUrl?, fileUrl?, meta:{serverStatusHint} }`。
- `serverStatusHint` 语义：`0` 放行输入、`1` 禁用输入（忙）。前端**只认 `text` 非空才渲染**，所以“忙”提示可以发 `{type:"server", meta:{serverStatusHint:1}}`（不带 text）。

> ⚠️ 注意：Java 后端里**根本没有这段 STOMP 端点的实现**（grep 零命中）。前端在等一个尚不存在的后端接口。Go 版必须**新建** `handler/websocket_handler.go` + STOMP 层，这部分是「构建」不是「复刻」。

---

## 四、5 个 Java 文件逐行映射

### 4.1 `MyManusApplication.java` → `main.go`

```java
@SpringBootApplication
public class MyManusApplication {
    public static void main(String[] args) {
        SpringApplication.run(MyManusApplication.class, args);
    }
}
```

```go
// main.go
func main() {
    cfg := config.Load()                    // 读 resources/application.yaml
    // 构建 Eino 模型 + Agent 引擎（见 §5）
    // 注册 STOMP 端点（见 §三）
    r := gin.Default()
    r.GET("/bs-dialog-websocket", handler.WebSocketHandler(cfg))
    r.Static("/file", cfg.File.Base)        // 静态文件服务
    r.Run(":" + cfg.Server.Port)            // 18081
}
```

### 4.2 `ModelConfig.java` → `config/model_config.go`（Eino ChatModel 工厂）

**Java 关键逻辑**（`buildByConfig`）：
```java
var bd = OpenAiApi.builder()
        .baseUrl(cfg.getBaseUrl())                 // https://ark.cn-beijing.volces.com/api/
        .apiKey(new SimpleApiKey(cfg.getApiKey()));
Optional.ofNullable(cfg.getCompletionsPath())
        .filter(StrUtil::isNotBlank)
        .ifPresent(bd::completionsPath);           // /v3/chat/completions
var option = OpenAiChatOptions.builder().model(cfg.getModelName()).build();
return OpenAiChatModel.builder().openAiApi(bd.build()).defaultOptions(option).build();
```

**端点结论**：`baseUrl + completionsPath = https://ark.cn-beijing.volces.com/api/v3/chat/completions`。

**Go 对应（Eino）**：
```go
package config

import openai "github.com/cloudwego/eino-ext/components/model/openai"

// BaseModelConfig 对应 Java ModelConfig.BaseModelConfig（@Data）
type BaseModelConfig struct {
    BaseURL         string `mapstructure:"base-url"`         // 对应 baseUrl
    APIKey          string `mapstructure:"api-key"`          // 对应 apiKey
    ModelName       string `mapstructure:"model-name"`       // 对应 modelName
    CompletionsPath string `mapstructure:"completions-path"` // 对应 completionsPath
    Stream          bool   `mapstructure:"stream"`           // Java 绑定但未使用，Go 可忽略
}

// NewChatModel 对应 buildByConfig()
// 两个 Bean：mainAgentModel()（agent-model 前缀，deepseek-v3）与 planModel()（plan-model 前缀，deepseek-r1）
func NewChatModel(ctx context.Context, cfg BaseModelConfig) (*openai.ChatModel, error) {
    return openai.NewChatModel(ctx, &openai.ChatModelConfig{
        BaseURL: resolveBaseURL(cfg),
        APIKey:  cfg.APIKey,
        Model:   cfg.ModelName,
        // 字段名以 pin 定的 eino-ext 版本为准（BaseURL/Model 等为示意）
    })
}

// resolveBaseURL 复刻 Java 拼接：baseUrl + completionsPath
//   "https://ark.cn-beijing.volces.com/api/" + "/v3/chat/completions"
// = "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
// Eino OpenAI 组件内部会追加 "/chat/completions"，故取前缀 "https://ark.cn-beijing.volces.com/api/v3"
func resolveBaseURL(cfg BaseModelConfig) string {
    base := strings.TrimRight(cfg.BaseURL, "/")
    path := strings.Trim(cfg.CompletionsPath, "/")
    path = strings.TrimSuffix(path, "chat/completions")
    path = strings.Trim(path, "/")
    if path == "" {
        return base
    }
    return base + "/" + path
}
```

> 注：Eino 组件如何处理 `BaseURL` 与 `/chat/completions` 的拼接，需在 `go get` 拉取后核对该组件的 README/源码再定稿；上面 `resolveBaseURL` 的目标语义（完整端点 = `.../api/v3/chat/completions`）是确定的。

### 4.3 `Constant.java` → `internal/constant/constant.go`

```java
public interface Constant {
    String TASK = "task";
    String AGENT_DATA = "agentData";
    String MAX_ACTIONS = "max_actions";
    interface Prompts {
        String AMAP_SYSTEM = "promptAmapSystem";
        String BROWSER_SYSTEM = "promptBrowserSystem";
        String BROWSER_USER_TASK = "promptBrowserUserTask";
        String CHART = "promptChart";
        String EXTRA_PAGE_CONTENT = "promptExtraPageContent";
        String HTML_DOC = "promptHtmlDoc";
        String PAGE_STATUS = "promptPageStatus";
        String PLANNING_STATUS = "promptPlanningStatus";
        String PLANNING_SYSTEM = "promptPlanningSystem";
        String PLANNING_TASK_MERGING = "promptPlanningTaskMerging";
        String PLANNING_USER_TASK = "promptPlanningUserTask";
        String TABLE = "promptTable";
        String VISION_EXTRACT = "promptVisionExtract";
    }
}
```

```go
package constant

const (
    Task       = "task"
    AgentData  = "agentData"
    MaxActions = "max_actions"
)

// Prompt 名称常量（对应 Constant.Prompts）
const (
    PromptAmapSystem          = "promptAmapSystem"
    PromptBrowserSystem       = "promptBrowserSystem"
    PromptBrowserUserTask     = "promptBrowserUserTask"
    PromptChart               = "promptChart"
    PromptExtraPageContent    = "promptExtraPageContent"
    PromptHtmlDoc             = "promptHtmlDoc"
    PromptPageStatus          = "promptPageStatus"
    PromptPlanningStatus      = "promptPlanningStatus"
    PromptPlanningSystem      = "promptPlanningSystem"
    PromptPlanningTaskMerging = "promptPlanningTaskMerging"
    PromptPlanningUserTask    = "promptPlanningUserTask"
    PromptTable               = "promptTable"
    PromptVisionExtract       = "promptVisionExtract"
)
```

### 4.4 `AgentTypeEnum.java` → `internal/enum/agent_type.go`

Java 枚举的 `agentName` 与 `desc`（**原文，1:1 复刻**）：

| agentName | desc |
|-----------|------|
| `reActPlanningAgent` | 任务规划智能体 |
| `BrowserAgent` | 浏览器Agent可以进行通用浏览器操作，例如通过网站查询到需要的信息或是进行指定的网页操作 |
| `TableAgent` | 此Agent专职用于绘制表格，只能基于上下文中已有的数据进行绘制，无法查询额外信息 |
| `ChartAgent` | 此Agent专职用于绘制统计图，只能基于上下文中已有的数据进行绘制，无法查询额外信息 |
| `HtmlDocAgent` | 此Agent用于生成各类网页内容，只能基于上下文中已有的数据进行生成，无法查询额外信息；可作为生成一般内容时的默认Agent |
| `AMAPAgent` | 此Agent包含完整的地图工具集，可用于路线规划、结构化地址转换为经纬度坐标等地理信息操作，返回文字或多媒体链接的结果 |

```go
package enum

type AgentType struct {
    AgentName string
    Desc      string
}

var (
    ReActPlanningAgent = AgentType{"reActPlanningAgent", "任务规划智能体"}
    BrowserAgent       = AgentType{"BrowserAgent", "浏览器Agent可以进行通用浏览器操作，例如通过网站查询到需要的信息或是进行指定的网页操作"}
    TableAgent         = AgentType{"TableAgent", "此Agent专职用于绘制表格，只能基于上下文中已有的数据进行绘制，无法查询额外信息"}
    ChartAgent         = AgentType{"ChartAgent", "此Agent专职用于绘制统计图，只能基于上下文中已有的数据进行绘制，无法查询额外信息"}
    HtmlDocAgent       = AgentType{"HtmlDocAgent", "此Agent用于生成各类网页内容，只能基于上下文中已有的数据进行生成，无法查询额外信息；可作为生成一般内容时的默认Agent"}
    AMAPAgent          = AgentType{"AMAPAgent", "此Agent包含完整的地图工具集，可用于路线规划、结构化地址转换为经纬度坐标等地理信息操作，返回文字或多媒体链接的结果"}
)

var agentTypes = []AgentType{
    ReActPlanningAgent, BrowserAgent, TableAgent, ChartAgent, HtmlDocAgent, AMAPAgent,
}

// AgentNameOf 对应 AgentTypeEnum.agentNameOf()（hutool EnumUtil.getBy，未找到返回 null）
func AgentNameOf(agentName string) *AgentType {
    for i := range agentTypes {
        if agentTypes[i].AgentName == agentName {
            return &agentTypes[i]
        }
    }
    return nil
}
```

> 注：Java `toString()` 返回的是枚举名（如 `BROWSER_AGENT`），与 `agentName`（如 `BrowserAgent`）**不同**。目前 Java 无业务代码使用 `toString()`，Go 无需模拟枚举名；若后续发现前端/日志依赖枚举名，再补一个 `Name` 字段。

### 4.5 `JsonFinder.java` → `internal/util/json_finder.go`

Java 逻辑：从左到右找第一个 `{` 或 `[`，再向后找第一个匹配的 `}` 或 `]`，用 Jackson `readTree` 校验，返回第一个合法片段。

```go
package util

import "encoding/json"

// FindFirstJson 返回第一个合法 JSON 对象或数组（对应 findFirstJson）
func FindFirstJson(src string) (string, bool) {
    for i := 0; i < len(src); i++ {
        var start, end byte
        switch src[i] {
        case '{':
            start, end = '{', '}'
        case '[':
            start, end = '[', ']'
        default:
            continue
        }
        _ = start
        for j := i + 1; j < len(src); j++ {
            if src[j] == end {
                candidate := src[i : j+1]
                if isValidJson(candidate) {
                    return candidate, true
                }
            }
        }
    }
    return "", false
}

func isValidJson(s string) bool {
    var v any
    return json.Unmarshal([]byte(s), &v) == nil
}
```

> 说明：`{ } [ ]` 均为单字节 ASCII，Go 按字节遍历与 Java 按 UTF-16 code unit 遍历在此场景下等价；`json.Unmarshal` 对应 Jackson `readTree` 的校验语义。

---

## 五、ReAct 规划引擎（核心业务逻辑，Go 从零构建 + Eino 编排）

这部分**没有 Java 实现**，其“逻辑”完全由 `schemaBaseReAct.json` + `promptPlanningSystem.txt` 声明。下面把它还原成可执行规格。

### 5.1 真实输出契约（修正原方案 §6.3）

`schemaBaseReAct.json` 只形式化声明了 `done`，但 `promptPlanningSystem.txt` 明确 action 有 **两种**：

| action | 参数 | 用途 |
|--------|------|------|
| `generateNext` | `{agent, subTask, maxStep}` | 规划下一个子任务，交给指定 Agent |
| `done` | `{success, text}` | 结束规划，`text` 必须完整带上最后一个子任务的结果 |

**修正后的 Go struct**：
```go
package model

// AgentOutput 对应 schemaBaseReAct.json
type AgentOutput struct {
    CurrentState CurrentState `json:"current_state"`
    Action       []ActionItem `json:"action"` // prompt 规定每次仅 1 个 action
}

type CurrentState struct {
    EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
    Memory                 string `json:"memory"`
    Thinking               string `json:"thinking"`
}

type ActionItem struct {
    GenerateNext *GenerateNextAction `json:"generateNext,omitempty"`
    Done         *DoneAction         `json:"done,omitempty"`
}

type GenerateNextAction struct {
    Agent   string `json:"agent"`
    SubTask string `json:"subTask"`
    MaxStep int    `json:"maxStep"`
}

type DoneAction struct {
    Success bool   `json:"success"`
    Text    string `json:"text"`
}
```

### 5.2 规划循环规则（`promptPlanningSystem.txt` 提炼，务必逐条实现）

**输入（4 要素）**：
1. `[最终目标]`
2. `[可选Agent]` —— 6 个 Agent 的能力概述（来自 `AgentTypeEnum` 的 desc）
3. `[子任务链]` —— 格式 `[Agent, Task:..., Result:...] -> [Agent, Task:..., Result:...] -> ...`，末尾是最近执行的子任务
4. `[最新已完成子任务结果]`

**输出**：`current_state`（evaluation_previous_goal / memory / thinking）+ 恰好 1 个 `action`。

**硬规则**：
- 子任务 `subTask` 用中文，独立且具体，保持抽象性，**不丢失用户原始要求的任何要素**；
- `maxStep` 最大不超过 16，参考已完成子任务的 `resultStep`（`-1` 表示不支持设置步数）；
- 子任务可能被部分完成，规划时要结合部分完成情况生成新子任务，**不重复执行已完成部分**；
- `done` 的 `text` 必须**完整**传最后一个子任务结果，禁止省略/总结/丢失数据；
- 规划 Agent **只负责拆解与规划，禁止用 `done` 直接完成任务**，必须把任务交给子 Agent；
- 不虚构 Agent，名称必须在可选 Agent 范围内；
- 浏览器指定站点必须是**中国大陆境内**站点。

### 5.3 Eino Graph 编排设计（示意）

```
[START]
   │
   ▼
planning（规划节点：调用 planModel + 规划 prompt，解析 AgentOutput）
   │
   ├─ action = generateNext ──► 按 action.agent 路由到子 Agent 节点
   │      ├─ BrowserAgent  → 执行 → 结果回填子任务链 → 回到 planning
   │      ├─ TableAgent    → 执行 → 结果回填子任务链 → 回到 planning
   │      ├─ ChartAgent    → 执行 → 结果回填子任务链 → 回到 planning
   │      ├─ HtmlDocAgent  → 执行 → 结果回填子任务链 → 回到 planning
   │      └─ AMAPAgent     → 执行 → 结果回填子任务链 → 回到 planning
   │
   └─ action = done ──► [END]，把 done.text 推给前端（serverStatusHint:0）
```

```go
// internal/agent/graph.go（示意）
import (
    "github.com/cloudwego/eino/flow/graph"
    "github.com/cloudwego/eino/schema"
)

func buildAgentGraph(ctx context.Context) (*graph.Graph[string, *schema.Message], error) {
    g := graph.NewGraph[string, *schema.Message]()
    _ = g.AddNode("planning", planningNode) // 调 planModel，解析 AgentOutput
    _ = g.AddNode("browser", browserNode)
    _ = g.AddNode("table", tableNode)
    _ = g.AddNode("chart", chartNode)
    _ = g.AddNode("htmldoc", htmldocNode)
    _ = g.AddNode("amap", amapNode)
    _ = g.AddEdge(graph.START, "planning")
    _ = g.AddBranch("planning", routeNext) // 返回 "browser"/"table"/.../graph.END
    return g.Compile(ctx)
}
```

> Eino Graph 只做**调度骨架**：节点内部仍严格按 §5.2 的 prompt + 规则执行，保证与声明式逻辑一致；**不套用 Eino 内置 ReAct 消息流**。

---

## 六、`application.yml` → Go 配置映射（含修正）

Java `application.yml`：
```yaml
server:
  port: 18081
spring:
  application:
    name: my-manus

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
  base: F:\\tmp\\my-manus\\
  domain: http://localhost:18081
```

Go 对应结构（Viper + mapstructure）：
```go
package config

type Config struct {
    Server     ServerConfig     `mapstructure:"server"`
    AgentModel BaseModelConfig  `mapstructure:"agent-model"`
    PlanModel  BaseModelConfig  `mapstructure:"plan-model"`
    File       FileConfig       `mapstructure:"file"`
}

type ServerConfig struct {
    Port int `mapstructure:"port"`
}

type FileConfig struct {
    Base   string `mapstructure:"base"`   // 读取后为 F:\tmp\my-manus\（双反斜杠是 YAML 转义）
    Domain string `mapstructure:"domain"`
}
```

关键点：
- `api-key: ${VOLCES_API_KEY}` → Go 里用 `os.Getenv("VOLCES_API_KEY")` 或 Viper 的 `${VOLCES_API_KEY}` 展开；
- `base-url` / `model-name`（kebab-case）→ Spring 松散绑定到 `baseUrl`/`modelName`，Go 里用 `mapstructure:"base-url"` 等显式对齐；
- `stream: false` 在 Java 中**未使用**，Go 里可以保留字段但不必实现流式逻辑。

---

## 七、Eino 组件映射汇总

| Java（spring-ai） | Eino（Go） | 说明 |
|-------------------|-----------|------|
| `OpenAiChatModel`（`spring-ai-openai`） | `github.com/cloudwego/eino-ext/components/model/openai` | `NewChatModel`，BaseURL/APIKey/Model 映射自 §4.2 |
| 提示词字符串（prompt/*.txt） | `github.com/cloudwego/eino/components/prompt` 的 `ChatTemplate` | 13 个模板加载 + 变量填充 |
| 规划循环调度（Java 未实现） | `github.com/cloudwego/eino/flow/graph` | 节点 = Agent，边 = 路由（§5.3） |
| MCP（`spring-ai-starter-mcp-client`） | `github.com/cloudwego/eino-ext/components/tool/mcp`（可选） | 暂无 MCP 使用点，可后置 |

---

## 八、遗留待确认项

1. **13 个 prompt 的注入时机**：`Constant.java` 只声明了名称，没有代码说明何时用哪个。需按 prompt 内容语义推断（如 `promptPlanningSystem` 是规划系统词、`promptPlanningStatus`/`promptPlanningTaskMerging`/`promptPlanningUserTask` 属于规划循环、`promptBrowserSystem`/`promptBrowserUserTask` 属于浏览器 Agent 等）。建议逐份读完 13 个 prompt 后，补一张「prompt → 调用点」映射表。
2. **子任务链 / resultStep 的数据结构**：只在 prompt 里出现（`[Agent, Task:..., Result:...]`、`resultStep`），Java 无 struct 定义，需在 Go 里自行设计内存结构。
3. **`file.base` 的实际用途**：只有 `application.yml` 声明了 `file:` 配置，前端 `Dialog.vue` 通过 `imageUrl`/`fileUrl` 展示，但没有 Java 代码生成文件，需在 Go 里按需实现文件落盘 + `/file/*` 静态服务。
4. **Eino 具体字段名**：`ChatModelConfig` / `Graph` 的精确 API 以 `go get` 后 pin 定的版本为准。

---

## 九、下一步

- 你确认本对照表后，我可以：
  1. 把 §二 的 4+1 处修正回写进 `JAVA_TO_GO_MIGRATION_PLAN.md`；
  2. 补齐「13 个 prompt → 调用点」映射（我逐份读完 prompt 后产出）；
  3. 开始生成可编译的 Go 代码骨架（`main.go` + `config/` + `internal/` + Eino Graph）。
