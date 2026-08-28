# MyManus Agent

MyManus 是一个多智能体（Multi-Agent）任务系统：用户通过浏览器对话框提交任务，后端由「任务规划智能体」将任务拆解为子任务，再调度浏览器 / 表格 / 图表 / 网页 / 高德地图等子智能体协作完成。

本项目是原 **Java 版**（`my-manus/`）到 **Go 版**（`my-manus-go/`）的 1:1 迁移，目标是**完美复刻原有业务逻辑**，前端保持不变。

---

## 目录结构

```
MyManus-Agent/
├── my-manus/                      # 【原 Java 项目】Spring Boot 3 + Vue3 前端
│   ├── pom.xml                    # Maven 依赖（spring-ai / websocket / playwright 等）
│   ├── src/main/java/cn/itcast/manus/   # 5 个 Java 源文件（骨架）
│   ├── src/main/resources/        # application.yml + schema + 13 个 prompt + js
│   └── my-manus-vue/              # 【前端，不动】Vue3 + @stomp/stompjs
│
├── my-manus-go/                   # 【Go 实现】本项目的核心交付物
│   ├── main.go                    # 入口（对应 MyManusApplication.java）
│   ├── config/                    # 配置 + 模型工厂（对应 ModelConfig.java）
│   ├── internal/
│   │   ├── constant/              # 常量 + 13 个 prompt 名称（Constant.java）
│   │   ├── enum/                  # 6 个 Agent 枚举（AgentTypeEnum.java）
│   │   ├── util/                  # JSON 提取工具（JsonFinder.java）
│   │   ├── model/                 # 消息 + AgentOutput 结构体
│   │   ├── ai/                    # DeepSeek OpenAI 兼容 HTTP 客户端
│   │   ├── prompt/                # 提示词加载 + 占位符填充
│   │   ├── stomp/                 # STOMP 子协议编解码
│   │   ├── handler/               # WebSocket + STOMP 端点（前端契约）
│   │   └── agent/                 # 规划引擎 + 6 个子 Agent
│   ├── resources/                 # application.yaml + schema + 13 prompt + js（原样复制）
│   └── testclient/                # STOMP 端到端测试客户端
│
├── JAVA_TO_GO_MIGRATION_PLAN.md   # 迁移方案（技术栈对比、实施步骤）
└── JAVA_TO_GO_MAPPING_DETAIL.md   # Java → Go 逐行对照表（基于真实源码）
```

---

## 技术栈

| 层 | Java 版 | Go 版 |
|---|---|---|
| Web 框架 | Spring Boot 3.4.3 | Gin v1.10 |
| WebSocket | spring-websocket (STOMP) | gorilla/websocket + 自实现 STOMP |
| AI 客户端 | spring-ai-openai | 标准库 `net/http`（OpenAI 兼容） |
| 配置 | application.yml + @ConfigurationProperties | Viper |
| 浏览器自动化 | Playwright | chromedp |
| 日志 | SLF4J | 标准库 `log` |

---

## 快速开始（Go 版）

### 前置条件

- Go 1.22+
- DeepSeek API Key（已配置在 `my-manus-go/resources/application.yaml`）
- 浏览器 Agent 需要本机安装 Chrome/Chromium
- 前端（可选）需要 Node.js，见「前端」一节

### 构建

```powershell
cd my-manus-go
go mod tidy
go build -o my-manus-go.exe .
```

### 运行

```powershell
.\my-manus-go.exe
# 输出：my-manus-go 监听 :18081
```

### 测试

另开一个终端，跑内置的 STOMP 测试客户端：

```powershell
go run ./testclient
# 或自定义问题：
go run ./testclient ws://localhost:18081/bs-dialog-websocket "帮我查一下今天北京天气"
```

期望流程：`CONNECT → CONNECTED → SUBSCRIBE → SEND → 规划 Agent 拆解 → 子 Agent 执行 → done → 返回结果`。

---

## 前端

前端无需任何改动，位于 `my-manus/my-manus-vue`。启动方式：

```powershell
cd my-manus/my-manus-vue
npm install
npm run dev
```

它会自动连接 `ws://localhost:18081/bs-dialog-websocket`（STOMP），订阅 `/user/queue/dialog`，向 `/app/enhanced-dialog` 发送用户消息。

---

## 架构

### 通信契约（前端 ↔ 后端）

```
ws://localhost:18081/bs-dialog-websocket
发送目标：/app/enhanced-dialog
订阅目标：/user/queue/dialog

发送格式：{"type":"user","text":"..."}
返回格式：{"type":"server","text":"...","imageUrl":"...","fileUrl":"...","meta":{"serverStatusHint":0}}
serverStatusHint：0 = 允许输入，1 = 禁用输入（服务器忙）
```

### ReAct 规划循环

1. 用户消息进入 `internal/agent/engine.go`；
2. `planModel`（deepseek-reasoner）输出 `AgentOutput`，其中 `action` 为二选一：
   - `generateNext {agent, subTask, maxStep}` → 调度到对应子 Agent；
   - `done {success, text}` → 结束，把 `text` 回给前端；
3. 子 Agent 执行结果回填「子任务链」，继续循环直到 `done`。

### 6 个智能体

| Agent | agentName | 职责 |
|---|---|---|
| 任务规划 | `reActPlanningAgent` | 任务拆解、子任务调度、结果合并 |
| 浏览器 | `BrowserAgent` | 通用浏览器操作、网页信息提取 |
| 表格 | `TableAgent` | 基于上下文数据生成 HTML 表格 |
| 图表 | `ChartAgent` | 基于上下文数据生成 HTML 图表 |
| 网页 | `HtmlDocAgent` | 生成 HTML 多媒体文档 |
| 高德地图 | `AMAPAgent` | 路线规划、地理编码等地图操作 |

---

## 配置

核心配置在 `my-manus-go/resources/application.yaml`：

```yaml
server:
  port: 18081

agent-model:      # 主 Agent 模型（deepseek-chat ≈ v3）
  base-url: https://api.deepseek.com/v1
  model-name: deepseek-chat

plan-model:       # 规划 Agent 模型（deepseek-reasoner ≈ r1）
  base-url: https://api.deepseek.com/v1
  model-name: deepseek-reasoner

file:             # 生成文件（HTML/图片）存储目录与访问域名
  base: ./file/
  domain: http://localhost:18081
```

> ⚠️ API Key 目前直接写在 yaml 里方便测试，上线前请改为环境变量 `${DEEPSEEK_API_KEY}`。

---

## 已知边界 / TODO

1. **浏览器 Agent**：已打通 `goToUrl` / `extractContent` / `wait` / `done` 核心闭环；`inputText` / `clickElement` / `switchTab`（依赖 `buildDomTree.js` 的可交互元素索引）待接入。
2. **高德地图 Agent**：工具链（地理编码 / 路线规划等）留了接入点，当前先由模型自身知识回答。
3. **MCP**：原 Java 有 `spring-ai-starter-mcp-client`，Go 版暂未接入 MCP。
4. **文件路径**：`file.base` 由 Java 的 `F:\tmp\my-manus\` 改为相对路径 `./file/`，便于本地运行。

---

## 迁移文档

- [`JAVA_TO_GO_MIGRATION_PLAN.md`](./JAVA_TO_GO_MIGRATION_PLAN.md) —— 整体迁移方案、技术栈对比、实施步骤。
- [`JAVA_TO_GO_MAPPING_DETAIL.md`](./JAVA_TO_GO_MAPPING_DETAIL.md) —— 基于真实源码的 Java → Go 逐行对照表（含 4+1 处修正）。

---

## Git

```powershell
git add .
git commit -m "feat: my-manus Java → Go 迁移（STOMP + ReAct 引擎 + 6 Agent + DeepSeek）"
```
